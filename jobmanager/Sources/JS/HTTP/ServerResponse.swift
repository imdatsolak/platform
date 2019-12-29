//
//  ServerResponse.swift
//  JobServer
//
//  Created by Dmytro Naumov on 23.05.18.
//  Copyright © 2018 Dmytro Naumov. All rights reserved.
//

import Foundation
import NIO
import NIOHTTP1


open class ServerResponse {

    public var status = HTTPResponseStatus.ok
    public var headers = HTTPHeaders()
    public let channel: Channel
    private var didWriteHeader = false
    private var didEnd = false

    public init(channel: Channel) {
        self.channel = channel
    }

    /// An Express like `send()` function.
    open func send(_ s: String) {
        flushHeader()

        let utf8 = s.utf8
        var buffer = channel.allocator.buffer(capacity: utf8.count)
        buffer.write(bytes: utf8)

        let part = HTTPServerResponsePart.body(.byteBuffer(buffer))

        _ = channel.writeAndFlush(part)
            .mapIfError(handleError)
            .map { self.end() }
    }

    /// Check whether we already wrote the response header.
    /// If not, do so.
    func flushHeader() {
        guard !didWriteHeader else { return } // done already
        didWriteHeader = true

        let head = HTTPResponseHead(version: .init(major: 1, minor: 1), status: status, headers: headers)
        let part = HTTPServerResponsePart.head(head)
        _ = channel.writeAndFlush(part).mapIfError(handleError)
    }

    func handleError(_ error: Error) {
        Logger.e("ERROR:", error)
        end()
    }

    func end() {
        guard !didEnd else { return }
        didEnd = true
        _ = channel.writeAndFlush(HTTPServerResponsePart.end(nil)).map { self.channel.close() }
    }
}

//JSON response
public extension ServerResponse {
    func json<T: Encodable>(_ model: T) {
        let data: Data
        do {
            data = try JSONEncoder().encode(model)
        }
        catch {
            return handleError(error)
        }

        self.sendData(data)
    }
}

public extension ServerResponse { //raw data response
    func sendData(_ data: Data) {

        self["Contgent-Type"] = "application/json"
        self["Content-Length"] = "\(data.count)"

        flushHeader()
        var buffer = channel.allocator.buffer(capacity: data.count)
        buffer.write(bytes: data)
        let part = HTTPServerResponsePart.body(.byteBuffer(buffer))
        _ = channel.writeAndFlush(part).mapIfError(handleError).map { self.end() }
    }
}

public extension ServerResponse {
    func sendJsonFrom(dictionary: [String: Any]) {

        let data: Data
        do {
            data = try JSONSerialization.data(withJSONObject: dictionary, options: [])
        }
        catch {
            return handleError(error)
        }

        self["Contgent-Type"] = "application/json"
        self["Content-Length"] = "\(data.count)"

        flushHeader()
        var buffer = channel.allocator.buffer(capacity: data.count)
        buffer.write(bytes: data)
        let part = HTTPServerResponsePart.body(.byteBuffer(buffer))
        _ = channel.writeAndFlush(part).mapIfError(handleError).map { self.end() }
    }
}

public extension ServerResponse {
    public subscript(name: String) -> String? {
        set {
            assert(!didWriteHeader, "Header is not written")
            if let v = newValue {
                headers.replaceOrAdd(name: name, value: v)
            }
                else {
                    headers.remove(name: name)
            }
        }
        get {
            return headers[name].joined(separator: ", ")
        }
    }
}

public extension ServerResponse {
    public static func respondWithError(ctx: ChannelHandlerContext, status: HTTPResponseStatus, message: String?) {
        Logger.w("Respond with error: \(message ?? "No message was provided")")
        let response = ServerResponse(channel: ctx.channel)
        response.status = status
        response.send(message ?? "Unknown error")
    }
}
