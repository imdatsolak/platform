//
//  JobManager.swift
//  JobServer
//
//  Created by Dmytro Naumov on 28.05.18.
//  Copyright © 2018 Dmytro Naumov. All rights reserved.
//

import Foundation
import NIO
import NIOHTTP1


//==================================================================
/// Used for one time routes setupe for each service
class RoutesConfig {

    static let API_VER = "1.0"
    //job ppathes
    static let jobPathPrefix: String = "/" + API_VER + "/" // + "job/"
    static let jobCreatePath: String = jobPathPrefix + "new-job"
    static let jobStatusPath: String = jobPathPrefix + "status"
    static let jobResultPath: String = jobPathPrefix + "result"
    static let jobDeletePath: String = jobPathPrefix + "delete"

    static let uploadDonePath: String = jobPathPrefix + "file-uploaded"


    private static let shared = RoutesConfig()
    public class func sharedInstance() -> RoutesConfig {
        return shared
    }

    init() {
        let mainServer = server
        self.setupJobCommonRoutes(for: mainServer)
        setupObjectDetectionRoutes(for: mainServer)
    }

    private func setupJobCommonRoutes(for server: Server) {
        //Create job
        server.post(RoutesConfig.jobCreatePath) { req, res, _ in
            createJob(from: req, response: res)
        }

        //get status
        server.post(RoutesConfig.jobStatusPath) { req, res, _ in
            sendJobStatus(from: req, response: res, job: nil)
        }

        //results
        server.post(RoutesConfig.jobResultPath) { req, res, _ in
            sendJobResult(from: req, response: res)
        }
    }

    //Object detection routes
    //This requests are coming from object detection instance (Detectron)
    //and handled in Object detection file
    private func setupObjectDetectionRoutes(for server: Server) {

    }
}

//==================================================================
//MARK: - Route methods
fileprivate func createJob(from request: ServerRequest, response: ServerResponse) {
    Logger.i("--= Create job =--")

    // 10) cretae job instance
    guard let job = request.mapJsonTo(type: Job.self) else {
        response.status = .badRequest
        response.send("Job not created. Please check your JSON")
        return
    }

    guard serviceManager.supportService(serviceId: job.targetService) else {
        response.status = .badRequest
        response.send("Service is not supported right now. Either it's down or it is wrong service ID")
        return
    }
    let jobIsAsync = serviceManager.asyncJobIds.contains(job.targetService)
    if jobIsAsync {
        // 20) save job to DB
        _ = writeJobToDB(job: job)
        // 30) else return job status
        response.status = httpStatusAccepted
        job.jobStatus = .created
        sendJobStatus(from: request, response: response, job: job)

        // 40) and start async service
        startJobProcessing(job: job)
    } else {
        // 50) if job is sync - do it and return result
        doSyncJob(job: job, response: response)
    }
}

fileprivate func sendJobStatus(from request: ServerRequest, response: ServerResponse, job: Job?) {
    Logger.enter()
    if job == nil {
        Logger.i("--= Job status =--")
    }
    var shouldDelete = false
    guard let jobStatusRequest = request.mapJsonTo(type: JobStatusRequest.self) else {
        response.status = .badRequest
        response.sendJsonFrom(dictionary: ["error": "Wrong JSON. Please check your parameters. Job ID is Int value"])
        Logger.exit()
        return
    }

    var jobForResponse = job

    if jobForResponse == nil {
        jobForResponse = getJobFromDB(jobId: jobStatusRequest.jobId!)
        if jobForResponse?.jobStatus == .ERROR {
            shouldDelete = true
        }
    }

    guard jobForResponse != nil else {
        response.status = httpStatusResourceNotFound
        response.sendJsonFrom(dictionary: ["error": "Job not found"])
        Logger.exit()
        return
    }

    let respDict = jobForResponse!.statusResponseDictionary()

    var statusCode = 200
    switch jobForResponse!.jobStatus {
    case .running, .created, .waitingForFile:
        statusCode = 202
    case .done, .ERROR:
        statusCode = jobForResponse?.jobResult?.httpResponseCode ?? 505
    default:
        statusCode = 505
    }

    response.status = HTTPResponseStatus(statusCode: statusCode)
    response.sendJsonFrom(dictionary: respDict)
    if shouldDelete { //if job was finished with error - we remove job from DB after status request
        _ = deleteJob(jobId: jobForResponse!.jobId!)
    }
    Logger.exit()
}

fileprivate func sendJobResult(from request: ServerRequest, response: ServerResponse) {
    Logger.i("--= Job results =--")
    request.jsonRead()

    guard let jobResultRequest = request.mapJsonTo(type: JobStatusRequest.self) else {
        response.status = .badRequest
        response.sendJsonFrom(dictionary: ["error": "Wrong JSON. Please check your parameters. Job ID is Int value"])
        return
    }

    guard let jobForResponse = getJobFromDB(jobId: jobResultRequest.jobId!) else {
        response.status = httpStatusResourceNotFound
        response.sendJsonFrom(dictionary: ["error": "Job not found"])
        return
    }

    guard jobForResponse.jobStatus == .done else {
        response.status = httpStatusResourceNotFound
        response.sendJsonFrom(dictionary: ["error": "Results not found. Job not finished yet"])
        return
    }

    if jobForResponse.jobResult!.succeed {
        if let dataToSend = jobForResponse.jobResult?.stringResult?.data(using: .utf8) {
            response.status = .ok
            response.sendData(dataToSend)
            _ = deleteJob(jobId: jobResultRequest.jobId!)
        }
            else {
                response.status = .internalServerError
                response.sendJsonFrom(dictionary: ["error": "Result decoding error"])
                return
        }
    }
        else {
            response.status = httpStatusResourceNotFound
            response.status = httpStatusResourceNotFound
            response.sendJsonFrom(dictionary: ["error": "Result not found. Job handling error. Please start job again"])
    }
}

//==================================================================
//MARK: - Async services

fileprivate func startJobProcessing(job: Job) {
    guard let jobPayload = job.payload else {
        job.jobStatus = .ERROR
        let jobResult = AIServiceResult()
        jobResult.succeed = false
        jobResult.message = "Payload for service not provided"
        job.jobResult = jobResult
        _ = writeJobToDB(job: job)
        return
    }

    job.jobStatus = .running
    _ = writeJobToDB(job: job)

    let jobService = JobService(with: jobPayload)
    jobService.start(jobId: job.jobId!, targetService: job.targetService, completion: { result in
        job.jobResult = result
        job.jobStatus = result.succeed ? .done : .ERROR
        _ = writeJobToDB(job: job)
        Logger.i("Service for job: \(job.jobId!) done " + (result.succeed ? "successfully" : "with error: \(result.message)"))
    })
}

//==================================================================
//MARK: - Sync services Methods

fileprivate func doSyncJob(job: Job, response: ServerResponse) {
    guard let jobPayload = job.payload else {
        response.status = HTTPResponseStatus.badRequest
        response.sendJsonFrom(dictionary: ["error": "Payload for service not provided"])
        return
    }

    let jobService = JobService(with: jobPayload)

    jobService.start(jobId: job.jobId!, targetService: job.targetService, completion: { result in
        response.status = HTTPResponseStatus(statusCode: result.httpResponseCode)
        if let dataToSend = result.stringResult?.data(using: .utf8) {
            response.sendData(dataToSend)
        }
            else {
                response.status = .internalServerError
                response.sendJsonFrom(dictionary: ["error": "Result decoding error"])
                Logger.e("Can't parse job result: \(result.stringResult ?? "")")
                return
        }
        Logger.i("SYNC service for job: \(job.jobId!) done " + (result.succeed ? "successfully" : "with error: \(result.message)"))
    })
}

//==================================================================
//MARK: - DB Methods

//our dummy 'database'
var jobList = [Int: Job]()

fileprivate func getJobFromDB(jobId: Int) -> Job? {
    let jobRes = DbProvider().getJob(jobId: jobId)
    return jobRes
}

fileprivate func writeJobToDB(job: Job) -> Bool {
    var writeSucceed = false
    let dbProvider = DbProvider()
    writeSucceed = dbProvider.saveJob(job: job)
    if !writeSucceed {
        Logger.e("Error during job saving into db. JobID: \(String(describing: job.jobId))")
    }
    return writeSucceed
}

fileprivate func deleteJob(jobId: Int) -> Bool {
    var deleteOk = false
    let dbProvider = DbProvider()
    deleteOk = dbProvider.deleteJob(jobId: jobId)
    if !deleteOk {
        Logger.e("Error during job removing from db. JobID: \(jobId)")
    }
    return deleteOk
}


