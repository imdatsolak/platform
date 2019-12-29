//
//  QueryString.swift
//  JobServer
//

import Foundation

fileprivate let paramDictKey = "queryParams"

public func querystring( req: ServerRequest, res: ServerResponse, next: @escaping Next) {
    if let queryItems = URLComponents(string: req.header.uri)?.queryItems {
        req.userInfo[paramDictKey] = Dictionary(grouping: queryItems, by: { $0.name }).mapValues { $0.compactMap({ $0.value }).joined(separator: ",") }
    }
    next()
}

public extension ServerRequest {
    func param(_ id: String) -> String? {
        return (userInfo[paramDictKey] as? [String: String])?[id]
    }
}
