//
//  DbProtocols.swift
//  CNIOAtomics
//

import Foundation

public protocol CommonDbProvider {

    func saveJob(job: Job) -> Bool
    func getJob(jobId: Int) -> Job?
    func deleteJob(jobId: Int) -> Bool
}
