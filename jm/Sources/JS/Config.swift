//
//  Config.swift
//  JobServer
//

import Foundation
import NIO
import NIOHTTP1

class SelfServerConfiguration: Codable {
    //server
    var host: String? = nil
    var port: Int = 9000
    var backlog: Int = 256
    var eventLoopGroup: EventLoopGroup? = nil

    enum CodingKeys: String, CodingKey {
        case host = "host"
        case port = "port"
    }

    init() {
        self.host = "127.0.0.1"
    }

    public required init(from decoder: Decoder) throws {
        let valueContainer = try decoder.container(keyedBy: CodingKeys.self)
        if let host = try? valueContainer.decode(String.self, forKey: CodingKeys.host) {
            self.host = host
        }
            else {
                print("Error during self server config initialisation")
        }

        if let port = try? valueContainer.decode(Int.self, forKey: CodingKeys.port) {
            self.port = port
        }
            else {
                print("Error during self server config initialisation")
        }
    }
}

class MySqlDbConfiguration: Codable {
    var host = ""
    var user = ""
    var password = ""
    var dbName = ""
    var tableName = ""

    enum CodingKeys: String, CodingKey {
        case host = "host"
        case user = "username"
        case password = "password"
        case dbName = "database_name"
        case tableName = "table_name"
    }
}

struct ServiceDiscoveryConfig: Codable {
    var serviceServerUrl: String = ""

    enum CodingKeys: String, CodingKey {
        case serviceServerUrl = "service_discovery_url"
    }
}

struct LogConfig: Codable {
    var consoleLogLevel = Loglevel.debug
    var fileLogLevel = Loglevel.debug
    var useFile = true
    var useConsole = true
    var logFilename = "JobServerLog.txt"

    enum CodingKeys: String, CodingKey {
        case consoleLogLevel = "console_log_level"
        case fileLogLevel = "file_log_level"
        case useFile = "use_file"
        case useConsole = "use_console"
        case logFilename = "log_filename"
    }
}


class JobServerConfig: Codable {
    var logConfig: LogConfig
    var mysqlConfig: MySqlDbConfiguration
    var serverConfig: SelfServerConfiguration
    var serviceDiscoveryConfig: ServiceDiscoveryConfig

    enum CodingKeys: String, CodingKey {
        case logConfig = "log_config"
        case mysqlConfig = "mysql_config"
        case serverConfig = "server_config"
        case serviceDiscoveryConfig = "service_discovery_config"
    }

    init() {
        self.logConfig = LogConfig()
        self.mysqlConfig = MySqlDbConfiguration()
        self.serverConfig = SelfServerConfiguration()
        self.serviceDiscoveryConfig = ServiceDiscoveryConfig()
    }

    public required convenience init(from decoder: Decoder) throws {
        self.init()

        let valueContainer = try decoder.container(keyedBy: CodingKeys.self)
        self.logConfig = try valueContainer.decode(LogConfig.self, forKey: CodingKeys.logConfig)
        self.mysqlConfig = try valueContainer.decode(MySqlDbConfiguration.self, forKey: CodingKeys.mysqlConfig)
        self.serverConfig = try valueContainer.decode(SelfServerConfiguration.self, forKey: CodingKeys.serverConfig)
        self.serviceDiscoveryConfig = try valueContainer.decode(ServiceDiscoveryConfig.self, forKey: CodingKeys.serviceDiscoveryConfig)
    }
}

