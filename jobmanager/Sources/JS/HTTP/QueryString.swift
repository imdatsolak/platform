//
//  QueryString.swift
//  JobServer
//
//  Created by Dmytro Naumov on 23.05.18.
//  Copyright © 2018 Dmytro Naumov. All rights reserved.
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
