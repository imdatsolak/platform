//
//  DbProtocols.swift
//  CNIOAtomics
//
//  Created by Dmytro Naumov on 04.06.18.
//

import Foundation

public protocol CommonDbProvider {

    func saveJob(job: Job) -> Bool
    func getJob(jobId: Int) -> Job?
    func deleteJob(jobId: Int) -> Bool
}
