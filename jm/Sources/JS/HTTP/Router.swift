//
//  Router.swift
//  JobServer
//
//  Created by Dmytro Naumov on 23.05.18.
//  Copyright © 2018 Dmytro Naumov. All rights reserved.
//

import Foundation
import NIO
import NIOHTTP1

open class Router {

    /// The sequence of Middleware functions.
    private var middleware = [Middleware]()

    /// Add another middleware (or many) to the list
    open func use(_ middleware: Middleware...) {
        self.middleware.append(contentsOf: middleware)
    }

    public func get(_ path: String = "", middleware: @escaping Middleware) {
        use { req, res, next in
            guard req.header.method == .GET,
                (req.header.uri.hasPrefix(path + "/") || req.header.uri == path || req.header.uri.hasPrefix(path + "?")) else {
                    return next()
            }
            middleware(req, res, next)
        }
    }

    public func post(_ path: String = "", middleware: @escaping Middleware) {
        use { req, res, next in
            guard req.header.method == .POST,
                (req.header.uri.hasPrefix(path + "/") || req.header.uri == path || req.header.uri.hasPrefix(path + "?")) else {
                    return next()
            }
            Logger.d("Path handled: \(req.header.uri)")
            middleware(req, res, next)
        }
    }

    /// Request handler. Calls its middleware list
    /// in sequence until one doesn't call `next()`.
    func handle(req: ServerRequest, resp: ServerResponse, next upperNext: @escaping Next) {
        let stack = self.middleware
        guard !stack.isEmpty else { return upperNext() }

        var next: Next? = { ( args: Any...) in }
        var i = stack.startIndex
        next = { (args: Any...) in
            // grab next item from matching middleware array
            let middleware = stack[i]
            i = stack.index(after: i)

            let isLast = i == stack.endIndex
            middleware(req, resp, isLast ? upperNext : next!)
        }
        next!()
    }
}
