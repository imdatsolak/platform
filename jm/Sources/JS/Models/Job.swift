//
//  Job.swift
//  JobServer
//

import Foundation

/// Stub class for result. Each service has to subclass it and implement Codable for result class
public class AIServiceResult: Codable {
    var succeed: Bool = true
    var message: String = ""
    var stringResult: String? = ""
    var httpResponseCode: Int = -1

    enum CodingKeys: String, CodingKey {
        case stringResult = "string_result"
        case message = "message"
        case succeed = "succeed"
        case httpResponseCode = "http_response_code"
    }

    func stringJson() -> String {
        var result = ""
        if let dataResult = try? JSONEncoder().encode(self) {
            result = String(data: dataResult, encoding: String.Encoding.utf8)!
        }
        return result
    }

    public required convenience init(from decoder: Decoder) throws {
        self.init()
        let valueContainer = try decoder.container(keyedBy: CodingKeys.self)
        self.stringResult = (try valueContainer.decodeIfPresent(String.self, forKey: CodingKeys.stringResult)) ?? ""
        self.message = (try valueContainer.decodeIfPresent(String.self, forKey: CodingKeys.message)) ?? ""
        self.succeed = (try valueContainer.decodeIfPresent(Bool.self, forKey: CodingKeys.succeed)) ?? true
        self.httpResponseCode = (try valueContainer.decodeIfPresent(Int.self, forKey: CodingKeys.httpResponseCode)) ?? -1
    }
}

enum JobStatus: Int, Codable {
    case created = 0
    case waitingForFile = 101
    case running = 102
    case done = 103
    case GONE = 800
    case noAccess = 900
    case killed = 950
    case hanging = 960
    case NONE = 999
    case ERROR = 9999
}

public class Job: Codable {
    var jobId: Int?
    var jobStatus: JobStatus
    var targetService: Int = 0
    var uploadId: String?
    var requestDetails: String? = ""
    var payload: String? = nil
    var jobResult: AIServiceResult? = nil
    var appId: Int = -1
    var appInstanceId: Int = -1

    init() {
        self.jobId = -1
        self.jobStatus = .created
    }

    public func statusResponseDictionary() -> [String: Any] {
        var result = [String: Any]()
        result[CodingKeys.jobId.stringValue] = self.jobId!
        result[CodingKeys.jobStatus.stringValue] = self.jobStatus.rawValue
        if let jobRes = self.jobResult {
            if !jobRes.succeed {
                result["Error"] = jobRes.message
            }
        }
        return result
    }

    public func resultResponseDictionary() -> [String: Any] {
        var result = [String: Any]()
        result[CodingKeys.jobId.stringValue] = self.jobId!
        result[CodingKeys.jobStatus.stringValue] = self.jobStatus.rawValue
        return result
    }

    enum CodingKeys: String, CodingKey {
        case jobId = "job_id"
        case requestDetails = "request_details"
        case payload = "payload"
        case uploadId = "upload_identifier"
        case targetService = "job_type"
        case jobStatus = "job_status"
        case jobResult = "job_result"
        case appId = "application_id"
        case appInstanceId = "application_instance_id"
    }

    public required convenience init(from decoder: Decoder) throws {
        self.init()
        let valueContainer = try decoder.container(keyedBy: CodingKeys.self)
        self.jobId = (try valueContainer.decodeIfPresent(Int.self, forKey: CodingKeys.jobId)) ?? self.jobId
        self.appId = (try valueContainer.decodeIfPresent(Int.self, forKey: CodingKeys.appId)) ?? self.appId
        self.appInstanceId = (try valueContainer.decodeIfPresent(Int.self, forKey: CodingKeys.appInstanceId)) ?? self.appInstanceId
        self.uploadId = try valueContainer.decodeIfPresent(String.self, forKey: CodingKeys.uploadId)
        self.targetService = (try? valueContainer.decode(Int.self, forKey: CodingKeys.targetService)) ?? 0
        self.requestDetails = try valueContainer.decodeIfPresent(String.self, forKey: CodingKeys.requestDetails)
        self.jobStatus = (try JobStatus(rawValue: (valueContainer.decodeIfPresent(Int.self, forKey: CodingKeys.jobStatus)) ?? 0))!
        self.payload = (try? valueContainer.decode(String.self, forKey: CodingKeys.payload))

        //adding data which is needed by worker into payload json
        if self.payload != nil && self.uploadId != nil { //we have to add upload ID to payload - it will be used by service
            self.payload = self.payload!.addDictionaryToJsonString(dictToAdd: [CodingKeys.uploadId.rawValue: self.uploadId!])
            self.payload = self.payload!.addDictionaryToJsonString(dictToAdd: [CodingKeys.appId.rawValue: self.appId])
            self.payload = self.payload!.addDictionaryToJsonString(dictToAdd: [CodingKeys.appInstanceId.rawValue: self.appInstanceId])
        }

        if let resultJsonStr = try? valueContainer.decodeIfPresent(String.self, forKey: CodingKeys.jobResult),
            let resultData = resultJsonStr?.replacingOccurrences(of: "\n", with: "\\n").data(using: .utf8), //we have to escape \n for proper json initialisation
            let jobRes = try? JSONDecoder().decode(AIServiceResult.self, from: resultData) {
            self.jobResult = jobRes
        }
    }
}

struct JobStatusRequest: Codable {
    var jobId: Int?
    enum CodingKeys: String, CodingKey {
        case jobId = "job_id"
    }
}

let oneJob = Job.init()
