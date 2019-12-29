//
//  Utils.swift
//  JobServer
//

import Foundation


public extension Data {
    func printAsJson() {
        let jsonObj = try? JSONSerialization.jsonObject(with: self, options: .allowFragments)
        if let json = jsonObj {
            Logger.d("JSON:\n" + String(describing: json) + "\n")
        }
    }

    func asJsonString() -> String {
        var result = "Not a json"
        let jsonObj = try? JSONSerialization.jsonObject(with: self, options: .allowFragments)
        if let json = jsonObj {
            result = String(describing: json)
        }
        return result
    }

    static func objectToJsonString(object: Any) -> String {
        var result = "Not a JSON"
        do {
            let data = try JSONSerialization.data(withJSONObject: object, options: .prettyPrinted)
            if let string = String(data: data, encoding: String.Encoding.utf8) {
                result = string
            }
        } catch {
            Logger.e(error)
        }
        return result
    }

}

