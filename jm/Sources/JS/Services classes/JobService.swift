//
//  JobService.swift
//  JobServer
//
//  Created by Dmytro Naumov on 18.06.18.
//

import Foundation
import NIO
import NIOHTTP1

class JobService {
    var jobPayload: String
    init(with payload: String) {
        self.jobPayload = payload
    }

    func start(jobId: Int, targetService: Int, completion: @escaping (_ result: AIServiceResult) -> Void) {
        DispatchQueue.global().async {
            let serviceResult = AIServiceResult()
            //check if service is available at all
            if serviceManager.supportService(serviceId: targetService) == false {
                serviceResult.succeed = false
                serviceResult.message = "Service is not supported at this moment"
                serviceResult.httpResponseCode = Int(HTTPResponseStatus.badRequest.code)
                completion(serviceResult)
                return
            }

            serviceManager.getNextFreeService(serviceId: targetService, completion: { service, error in
                if let serviceForWork = service {
                    guard let url = URL(string: serviceForWork.actionUrl) else {
                        serviceResult.succeed = false
                        serviceResult.message = "Wrong service server URL"
                        serviceResult.httpResponseCode = Int(HTTPResponseStatus.badRequest.code)
                        completion(serviceResult)
                        return
                    }

                    //create and run request to service
                    var request = URLRequest(url: url)
                    request.httpMethod = "POST"
                    request.setValue("application/json; charset=utf-8", forHTTPHeaderField: "Content-Type")
                    request.httpBody = self.jobPayload.data(using: .utf8)

                    URLSession.shared.configuration.timeoutIntervalForRequest = 60.0 * 60.0 //1hr
                    URLSession.shared.configuration.timeoutIntervalForResource = 60.0 * 60.0 //1hr

                    URLSession.shared.dataTask(with: request, completionHandler: { (data, response, error) in
                        serviceManager.makeServiceFree(service: serviceForWork)

                        if error != nil {
                            serviceResult.succeed = false
                            serviceResult.message = error!.localizedDescription
                            serviceResult.httpResponseCode = 404
                            Logger.e(error!.localizedDescription)
                            completion(serviceResult)
                            return
                        }
                        let responseCode = (response as? HTTPURLResponse)!.statusCode
                        serviceResult.httpResponseCode = responseCode

                        guard let data = data else {
                            serviceResult.succeed = false
                            serviceResult.message = "No data provided by service"
                            completion(serviceResult)
                            return
                        }

                        guard responseCode == 200 else {
                            serviceResult.succeed = false
                            if let strData = String(data: data, encoding: .utf8) {
                                serviceResult.message = strData
                            }
                                else {
                                    serviceResult.message = "Service provide no response message"
                            }
                            completion(serviceResult)
                            Logger.e("Error from service server: \(service!.actionUrl). Called URL: \(request.url!.absoluteString) Code: \(responseCode). Error: \(serviceResult.message)")
                            return
                        }

                        if let strData = String(data: data, encoding: .utf8) {
                            serviceResult.succeed = true
                            serviceResult.message = ""
                            serviceResult.stringResult = strData
                        }
                            else {
                                serviceResult.httpResponseCode = Int(HTTPResponseStatus.internalServerError.code)
                                serviceResult.succeed = false
                                serviceResult.message = "Can't decode service output: \(data.asJsonString())"
                                data.printAsJson()
                        }
                        completion(serviceResult)
                    }).resume()

                } else { //service management error hapenned
                    serviceResult.succeed = false
                    serviceResult.httpResponseCode = Int(HTTPResponseStatus.internalServerError.code)
                    completion(serviceResult)
                    if error != nil {
                        serviceResult.message = error!
                    } else {
                        serviceResult.message = "AI service management unknown error"
                    }
                    completion(serviceResult)
                }
            })
        }
    }

    deinit {
        Logger.d("Deinit: \(self)")
    }
}
