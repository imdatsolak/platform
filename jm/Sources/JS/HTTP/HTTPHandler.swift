//
//  HTTPHandler.swift
//  JobServer
//

import Foundation
import NIO
import NIOHTTP1


let httpStatusAccepted = HTTPResponseStatus.custom(code: 202, reasonPhrase: "ACCEPTED")
let httpStatusGone = HTTPResponseStatus.custom(code: 401, reasonPhrase: "GONE")
let httpStatusResourceNotFound = HTTPResponseStatus.custom(code: 444, reasonPhrase: "JOB/UPLOAD_ID NOT FOUND")
//let httpStatus = HTTPResponseStatus.custom(code: , reasonPhrase: "")

final class HTTPHandler: ChannelInboundHandler {
    typealias InboundIn = HTTPServerRequestPart

    let router: Router

    var request: ServerRequest? = nil
    var response: ServerResponse? = nil

    init(router: Router) {
        self.router = router
    }

    func channelRead(ctx: ChannelHandlerContext, data: NIOAny) {
        let reqPart = unwrapInboundIn(data)

        switch reqPart {
        case .head(let header):
            self.request = ServerRequest(header: header)
            self.response = ServerResponse(channel: ctx.channel)

        case .body (var body):
            guard self.request != nil else {
                return
            }

            if let utfData = body.readBytes(length: body.readableBytes) {
                let data = Data.init(utfData)
                self.request!.jsonData = Data.init(data)
            }

        case .end:
            guard self.request != nil else {
                ServerResponse.respondWithError(ctx: ctx, status: .internalServerError, message: "Request read error")
                return
            }

            let req = self.request!
            let resp = self.response!
            router.handle(req: req, resp: resp) {
                (items: Any...) in // the final handler
                resp.status = .notFound
                Logger.e("Not handled path: \(req.header.uri)")
                resp.send("Wrong API point")
            }
        }
    }

    func channelActive(ctx: ChannelHandlerContext) {
        Logger.d("Channel ready, client address: \(ctx.channel.remoteAddress?.description ?? "-")")
    }

    func channelInactive(ctx: ChannelHandlerContext) {
        Logger.d("Channel closed. \(ObjectIdentifier(self))")
    }

/// Called if an error happens. We just close the socket here.
    func errorCaught(ctx: ChannelHandlerContext, error: Error) {
        Logger.e("ERROR: \(error)")
        ctx.close(promise: nil)
    }
}


