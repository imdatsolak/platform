//
//  Express.swift
//  JobServer
//

import Foundation
import NIO
import NIOHTTP1

public typealias Next = ( Any...) -> Void
public typealias Middleware = ( ServerRequest, ServerResponse, @escaping Next) -> Void


open class Server: Router {
    let configuration: SelfServerConfiguration
    let eventLoopGroup: EventLoopGroup
    var serverChannel: Channel?

    init(configuration: SelfServerConfiguration = mainConfig.serverConfig) {
        self.configuration = configuration
        self.eventLoopGroup = configuration.eventLoopGroup ?? MultiThreadedEventLoopGroup.init(numberOfThreads: System.coreCount)
    }

    func listenAndWait() {
        listen()
        do { try serverChannel?.closeFuture.wait() }
        catch { Logger.e("ERROR: Failed to wait on server: \(error)") }
    }


    open func listen() {
        let bootstrap = makeBootstrap()

        do {
            let address: SocketAddress
            if let host = configuration.host {
                address = try SocketAddress
                    .newAddressResolving(host: host, port: configuration.port)
            }
                else {
                    var addr = sockaddr_in()
                    addr.sin_port = in_port_t(configuration.port).bigEndian
                    address = SocketAddress(addr, host: "*")
            }

            serverChannel = try bootstrap.bind(to: address).wait()
            if let addr = serverChannel?.localAddress {
                Logger.i("Server running on: \(addr)")
            }
                else {
                    Logger.e("Server reported no local address?")
            }
        }
        catch let error as NIO.IOError {
            Logger.e("Failed to start server, errno:", error.errnoCode, "\n", error.localizedDescription)
        }
        catch {
            Logger.e("ERROR: failed to start server:", type(of: error), error)
        }
    }

    func makeBootstrap() -> ServerBootstrap {
        let reuseAddrOpt = ChannelOptions.socket(SocketOptionLevel(SOL_SOCKET), SO_REUSEADDR)
        let bootstrap = ServerBootstrap(group: eventLoopGroup)
        // Specify backlog and enable SO_REUSEADDR for the server itself
        .serverChannelOption(ChannelOptions.backlog, value: Int32(configuration.backlog))
            .serverChannelOption(reuseAddrOpt, value: 1)

        // Set the handlers that are applied to the accepted Channels
        .childChannelInitializer { channel in
            channel.pipeline
                .configureHTTPServerPipeline(withErrorHandling: true)
                .then {
                    channel.pipeline.add(name: "JobServer", handler: HTTPHandler(router: self))
            }
        }

        // Enable TCP_NODELAY and SO_REUSEADDR for the accepted Channels
        .childChannelOption(ChannelOptions.socket(IPPROTO_TCP, TCP_NODELAY), value: 1)
            .childChannelOption(reuseAddrOpt, value: 1)
            .childChannelOption(ChannelOptions.maxMessagesPerRead, value: 1)
        return bootstrap
    }
}


