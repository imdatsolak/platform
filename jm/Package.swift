// swift-tools-version:4.0
//
//  Package.swift
//  JobServer
//
//
import PackageDescription

let package = Package(
    name: "JS",

    dependencies: [
            .package(url: "https://github.com/apple/swift-nio.git", from: "1.6.1"),
            .package(url: "https://github.com/apple/swift-package-manager.git", from: "0.1.0"),
            .package(url: "https://github.com/PerfectlySoft/Perfect-MySQL.git", from: "3.0.0")
    ],

    targets: [
            .target(name: "JS",
                    dependencies: [
                        "NIO",
                        "NIOHTTP1",
                        "PerfectMySQL",
                        "Utility"
                    ])
    ]
)
