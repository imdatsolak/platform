//
//  Log.swift
//  JobServer
//

import Foundation

enum Loglevel: Int, Codable {
    case debug = 30
    case info = 40
    case warning = 50
    case error = 60
}


enum LogIcon: String {
    case funcEnter = "↘️ ENTER  " //function entry
    case funcExit = "↗️ EXIT   " //function exit
    case debug = "🔶 DEBUG  " // debug
    case info = "🚹 INFO   " // info
    case warning = "⚠️ WARNING" // warning
    case error = "🛑 ERROR  " // error}
}

class Logger {
    static var dateFormat = "yyyy-MM-dd HH:mm:ss.SSS"
    static var dateFormatter: DateFormatter {
        let formatter = DateFormatter()
        formatter.dateFormat = dateFormat
        formatter.locale = Locale.current
        formatter.timeZone = TimeZone.current
        return formatter
    }
    static var logConfig = LogConfig()

    /// Print log msg with function entry icon (log level is debug)
    ///
    /// - Parameters:
    ///   - msg: msg to print
    class func enter( _ msg: Any..., filename: String = #file, line: Int = #line, funcName: String = #function) {
        Logger.logMessage(msg, logIcon: LogIcon.funcEnter, logLevel: .debug, filename: filename, line: line, funcName: funcName)
    }

    /// Print log msg with function entry icon (log level is debug)
    ///
    /// - Parameters:
    ///   - msg: msg to print
    class func exit( _ msg: Any..., filename: String = #file, line: Int = #line, funcName: String = #function) {
        Logger.logMessage(msg, logIcon: LogIcon.funcExit, logLevel: .debug, filename: filename, line: line, funcName: funcName)
    }


    /// Print debug message
    ///
    /// - Parameters:
    ///   - msg: message to print
    class func d( _ msg: Any..., filename: String = #file, line: Int = #line, funcName: String = #function) {
        Logger.logMessage(msg, logIcon: LogIcon.debug, logLevel: .debug, filename: filename, line: line, funcName: funcName)
    }

    /// Print info message
    ///
    /// - Parameters:
    ///   - msg: message to print
    class func i( _ msg: Any..., filename: String = #file, line: Int = #line, funcName: String = #function) {
        Logger.logMessage(msg, logIcon: LogIcon.info, logLevel: .info, filename: filename, line: line, funcName: funcName)
    }

    /// Print warning message
    ///
    /// - Parameters:
    ///   - msg: message to print
    class func w( _ msg: Any..., filename: String = #file, line: Int = #line, funcName: String = #function) {
        Logger.logMessage(msg, logIcon: LogIcon.warning, logLevel: .warning, filename: filename, line: line, funcName: funcName)
    }

    /// Print error message
    ///
    /// - Parameters:
    ///   - msg: message to print
    class func e( _ msg: Any..., filename: String = #file, line: Int = #line, funcName: String = #function) {
        Logger.logMessage(msg, logIcon: LogIcon.error, logLevel: .error, filename: filename, line: line, funcName: funcName)
    }

    class fileprivate func logMessage( _ msg: [Any], logIcon: LogIcon, logLevel: Loglevel, filename: String = #file, line: Int = #line, funcName: String = #function) {

        var msgString = ""
        for object in msg {
            msgString = msgString + " \(object)"
        }
        if msgString.count > 0 {
            msgString = " ---> " + msgString
        }

        msgString = "[\(Date().toString())][\(logIcon.rawValue)|\(funcName) | L:\(line) | \(URL(fileURLWithPath: filename).lastPathComponent)] \(msgString)"

        if Logger.logConfig.useConsole && logLevel.rawValue >= Logger.logConfig.consoleLogLevel.rawValue {
            print(msgString)
        }

        if Logger.logConfig.useFile && logLevel.rawValue >= Logger.logConfig.fileLogLevel.rawValue {
            printToFile(msg: msgString)
        }
    }

    fileprivate class func printToFile(msg: String) {
        //TODO: This is slow method. Please make somethng faster (maybe log server) as soon as logging functionality will be better defined
        let fm = FileManager()
        let filePath = fm.currentDirectoryPath + "/\(logConfig.logFilename)"
        guard fm.isWritableFile(atPath: filePath) || fm.createFile(atPath: filePath, contents: nil, attributes: nil),
            let fileHandle = FileHandle(forWritingAtPath: filePath),
            let data = "\(msg)\n".data(using: .utf8) else {
                print("[ERR] Unable to open file at \"\(filePath)\" to log event")
                return
        }
        defer {
            fileHandle.closeFile()
        }
        fileHandle.seekToEndOfFile()
        fileHandle.write(data)
        fileHandle.closeFile()
    }
}

fileprivate extension Date {
    func toString() -> String {
        return Logger.dateFormatter.string(from: self as Date)
    }
}
