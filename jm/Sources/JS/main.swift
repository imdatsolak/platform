//
//  main.swift
//  JobServer
//

import Foundation
import Utility
import NIO
import NIOHTTP1
//=========================================================
//MARK: - Config file read
var mainConfig = JobServerConfig()

fileprivate func initConfig(_ filename: String) {
    let fm = FileManager()
    var filePath = filename
    if filename.components(separatedBy: "/").count == 1 {
        filePath = fm.currentDirectoryPath + "/\(filename)"
    }
    let fileUrl = URL.init(fileURLWithPath: filePath, isDirectory: false)

    guard fm.fileExists(atPath: filePath),
        let fileStr = try? String.init(contentsOf: fileUrl, encoding: .utf8)
        else {
            print("Config initialise error!")
            exit(1)
    }

    do {
        mainConfig = try JSONDecoder().decode(JobServerConfig.self, from: fileStr.data(using: .utf8)!)
    } catch let err {
        print("ERROR! Config read error! \(err)")
        exit(1)
    }
}

//=========================================================
//MARK: - Argument parse
// The first argument is always the executable, drop it
let arguments = Array(ProcessInfo.processInfo.arguments.dropFirst())

let parser = ArgumentParser(usage: "<options>", overview: "This is what this tool is for")
let configFilename: OptionArgument<String> = parser.add(option: "--config", shortName: "-c", kind: String.self, usage: "Name of config file if it is in current dir or path to it")

func processArguments(arguments: ArgumentParser.Result) {
    if let fileName = arguments.get(configFilename) {
        initConfig(fileName)
    }
        else {
            print("Config filename is not provided")
            exit(1)
    }
}

do {
    let parsedArguments = try parser.parse(arguments)
    processArguments(arguments: parsedArguments)
}
catch let error as ArgumentParserError {
    print(error.description)
}
catch let error {
    print(error.localizedDescription)
}


//=========================================================
//MARK: - Server setup
Logger.i("\n\n\n=========================================================\n              STARTING NEW SERVER INSTANCE              \n=========================================================\n")

let serviceManager = ServiceManager.sharedInstance()
let server = Server()

let routesConfig = RoutesConfig.sharedInstance()
server.use(querystring)
server.listenAndWait()
