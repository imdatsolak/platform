//
//  MySqlDatabaseProvider.swift
//  JobServer
//

import Foundation
import PerfectMySQL

let mySqlDbConnection: MySQL? = connectToDB()
let dbConfig = mainConfig.mysqlConfig

struct JobTableKeys {
    static let jobId = "job_id"
    static let appId = "application_id"
    static let appInstanceId = "application_instance_id"
    static let requestDetails = "request_details"
    static let payload = "payload"
    static let uploadId = "upload_identifier"
    static let targetService = "job_type"
    static let jobStatus = "job_status"
    static let jobResult = "job_result"
}

func connectToDB() -> MySQL? {

    let mysql = MySQL() // Create an instance of MySQL to work with

    let connected = connect(db: mysql, dbConfig: dbConfig)

    guard connected else {
        Logger.e("MYSQL connection error: \(mysql.errorMessage())")
        return nil
    }
    Logger.i("DB connection set and configured")
    mysql.setOption(.MYSQL_SET_CHARSET_NAME, "utf8")
    return mysql
}

func connect(db: MySQL, dbConfig: MySqlDbConfiguration)->Bool{
    let connected = db.connect(host: dbConfig.host, user: dbConfig.user, password: dbConfig.password, db: dbConfig.dbName)
    return connected
}

class DbProvider: CommonDbProvider {

    var dbConnection: MySQL? = nil
    var canWork = false
    var tableName = dbConfig.tableName
    init() {
        self.dbConnection = mySqlDbConnection
        guard let _ = mySqlDbConnection else {
            Logger.e("---------- Database is not set up ------------")
            return
        }
        if mySqlDbConnection?.ping() == false {
            if connect(db: mySqlDbConnection!, dbConfig: dbConfig) == false {
                Logger.e("DB is not conneted!")
            }
        }

        //create table if not exist
        let sql = """
        CREATE TABLE IF NOT EXISTS \(tableName) (
            internal_id INT NOT NULL AUTO_INCREMENT,
            \(JobTableKeys.jobId) INT NOT NULL DEFAULT 0,
            \(JobTableKeys.appId) INT NOT NULL DEFAULT -1,
            \(JobTableKeys.appInstanceId) INT NOT NULL DEFAULT -1,
            \(JobTableKeys.targetService) INT NOT NULL DEFAULT 0,
            \(JobTableKeys.jobStatus) INT NOT NULL DEFAULT 0,
            \(JobTableKeys.uploadId) VARCHAR(50) NULL DEFAULT NULL,
            \(JobTableKeys.payload) MEDIUMTEXT NULL DEFAULT NULL,
            \(JobTableKeys.jobResult) MEDIUMTEXT NULL DEFAULT NULL,
            PRIMARY KEY (internal_id)
        );
        """
        guard dbConnection!.query(statement: sql) else {
            Logger.e(dbConnection!.errorMessage())
            return
        }
        self.canWork = true
    }

    fileprivate func dbHasJobWithId(jobId: Int) -> Bool {
        var result = false
        let sqlStr = """
                        SELECT \(JobTableKeys.jobId) FROM \(tableName) WHERE \(JobTableKeys.jobId) = "\(jobId)"
                    """

        let querySuccess = dbConnection!.query(statement: sqlStr)
        // make sure the query worked
        guard querySuccess else {
            Logger.e("Select from database error for job with ID: \(jobId)")
            return false
        }

        // Save the results to use during this session
        let qResult = dbConnection!.storeResults()
        result = qResult!.numRows() >= 1
        return result
    }

    func createJob(job: Job) -> Bool {
        guard self.canWork else {
            Logger.e("DB is not configurted!")
            return false
        }

        if self.dbConnection!.ping() == false {
            // connection lost
            if connect(db: self.dbConnection!, dbConfig: dbConfig) == false {
                Logger.e("DB is not conneted!")
            }
            return false
        }

        let jobResultStr = (job.jobResult?.stringJson() ?? "").replacingOccurrences(of: "\"", with: "\\\"").replacingOccurrences(of: "\\\\\"", with: "\\\\\\\"")
        let jobPayloadStr = (job.payload ?? "").replacingOccurrences(of: "\"", with: "\\\"")
        let jobUploadId = job.uploadId ?? ""

        let sqlStr = """
            INSERT INTO \(self.tableName) (\(JobTableKeys.jobId), \(JobTableKeys.appId), \(JobTableKeys.appInstanceId), \(JobTableKeys.targetService),\(JobTableKeys.jobStatus),\(JobTableKeys.uploadId),\(JobTableKeys.payload),\(JobTableKeys.jobResult))
            VALUES  ("\(job.jobId!)", "\(job.appId)", "\(job.appInstanceId)", "\(job.targetService)", "\(job.jobStatus.rawValue)", "\(jobUploadId)", "\(jobPayloadStr)", "\(jobResultStr)");
        """

        if !self.dbConnection!.query(statement: sqlStr){
            Logger.e("Database error. Can't save job with ID: \(job.jobId!)")
            Logger.e(dbConnection!.errorMessage())
            return false
        }
        return true
    }
    
    func updateJob(job: Job) -> Bool {
        guard self.canWork else {
            Logger.e("DB is not configurted!")
            return false
        }

        if self.dbConnection!.ping() == false {
            // connection lost
            if connect(db: self.dbConnection!, dbConfig: dbConfig) == false {
                Logger.e("DB is not conneted!")
            }
            return false
        }

        //more escaping for the god of escaping :) Job result already has escaped string. We have to escape this escaped string again
        let jobResultStr = (job.jobResult?.stringJson() ?? "").replacingOccurrences(of: "\"", with: "\\\"").replacingOccurrences(of: "\\\\\"", with: "\\\\\\\"")
        let jobPayloadStr = (job.payload ?? "").replacingOccurrences(of: "\"", with: "\\\"")
        let jobUploadId = job.uploadId ?? ""

        let sqlStr = """
            UPDATE \(self.tableName)
            SET \(JobTableKeys.targetService) = "\(job.targetService)",
                \(JobTableKeys.jobStatus) = "\(job.jobStatus.rawValue)",
                \(JobTableKeys.uploadId) = "\(jobUploadId)",
                \(JobTableKeys.payload) = "\(jobPayloadStr)",
                \(JobTableKeys.jobResult) = "\(jobResultStr)"
            WHERE \(JobTableKeys.jobId) = \(job.jobId!)
        """
        if !self.dbConnection!.query(statement: sqlStr){
            Logger.e("Database error. Can't update job with ID: \(job.jobId!)")
            Logger.e(dbConnection!.errorMessage())
            return false
        }
        return true
    }

    func saveJob(job: Job) -> Bool {
        guard self.canWork else {
            Logger.e("DB is not configurted!")
            return false
        }

        if self.dbConnection!.ping() == false {
            // connection lost
            if connect(db: self.dbConnection!, dbConfig: dbConfig) == false {
                Logger.e("DB is not conneted!")
            }
            return false
        }

        var result = true
        //1) if item exist -> update job
        if dbHasJobWithId(jobId: job.jobId!){
            result = self.updateJob(job: job)
        }
        else {//2) if not exist -> create new record
            result = self.createJob(job: job)
        }
        return result
    }

    func getJob(jobId: Int) -> Job? {
        guard self.canWork else {
            Logger.e("DB is not configurted!")
            return nil
        }

        if self.dbConnection!.ping() == false {
            // connection lost
            if connect(db: self.dbConnection!, dbConfig: dbConfig) == false {
                Logger.e("DB is not conneted!")
            }
            return nil
        }
        var result: Job? = nil

        let sqlStr = """
        SELECT \(JobTableKeys.jobId), \(JobTableKeys.targetService), \(JobTableKeys.jobStatus), \(JobTableKeys.uploadId), \(JobTableKeys.payload), \(JobTableKeys.jobResult)
        FROM \(tableName)
        WHERE \(JobTableKeys.jobId) = "\(jobId)"
        """

        let querySuccess = dbConnection!.query(statement: sqlStr)
        // make sure the query worked
        guard querySuccess else {
            Logger.e("Select from database error for job with ID: \(jobId)")
            return nil
        }



        // Save the results to use during this session
        let qResult = dbConnection!.storeResults()
        if  qResult!.numRows() >= 1 {
            //array with optional strings in order: [jobId, targetService, jobStatus, uploadId, payload, result]
            let dbResArr = qResult!.next()!
            //lets make string json for Job
            let jsonDict = [(JobTableKeys.jobId) : (Int(dbResArr[0]!)!),
                            (JobTableKeys.targetService) : (Int(dbResArr[1]!)!),
                            (JobTableKeys.jobStatus): (Int(dbResArr[2]!)!),
                            (JobTableKeys.uploadId) : (dbResArr[3]!),
                            (JobTableKeys.payload) : (dbResArr[4]!),
                            (JobTableKeys.jobResult) : (dbResArr[5]!)
                            ] as [String : Any]

            if let data = try? JSONSerialization.data(withJSONObject: jsonDict, options: []){
                result = try? JSONDecoder().decode(Job.self, from: data)
                if result == nil {
                    Logger.e("Can't decode DB read result:", jsonDict)
                }
            }
        }
        else {
            Logger.w("No item with ID: \(jobId) found in DB")
        }
        return result
    }

    func deleteJob(jobId: Int) -> Bool {
        guard self.canWork else {
            Logger.e("DB is not configurted!")
            return false
        }

        if self.dbConnection!.ping() == false {
            // connection lost
            if connect(db: self.dbConnection!, dbConfig: dbConfig) == false {
                Logger.e("DB is not conneted!")
            }
            return false
        }

        let sqlStr = """
            DELETE FROM \(self.tableName) WHERE \(JobTableKeys.jobId) = \(jobId)
        """
        if !self.dbConnection!.query(statement: sqlStr){
            Logger.e("Database error. Can't DELETE job with ID:", jobId)
            Logger.e(dbConnection!.errorMessage())
            return false
        }
        return true
    }
}

