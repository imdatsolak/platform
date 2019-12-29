//
//  AIService.swift
//  JobServer
//
//  Created by Dmytro Naumov on 15.06.18.
//

import Foundation
class AIService: Codable, Comparable {
    var serviceType: Int
    var actionUrl: String
    var requiresUpload: Bool
    var returnsDownload: Bool
    var isAsync: Bool
    var inUse: Bool = false

    init() {
        self.serviceType = -1
        self.actionUrl = ""
        self.requiresUpload = false
        self.returnsDownload = false
        self.isAsync = true
    }

    enum CodingKeys: String, CodingKey {
        case serviceType = "service_type"
        case requiresUpload = "service_requires_upload"
        case returnsDownload = "service_returns_download"
        case isAsync = "service_is_async"
        case actionUrl = "service_action_url"
    }

    static func == (lhs: AIService, rhs: AIService) -> Bool {
        return (lhs.serviceType == rhs.serviceType && lhs.actionUrl == rhs.actionUrl)
    }

    static func < (lhs: AIService, rhs: AIService) -> Bool {
        return lhs.serviceType < rhs.serviceType
    }
}
