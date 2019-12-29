//
//  String+Json.swift
//  JobServer
//
//  Created by Dmytro Naumov on 19.06.18.
//

import Foundation

public extension String {
    func addDictionaryToJsonString(dictToAdd: [String: Any]) -> String? {
        var result: String? = nil
        if let data = self.data(using: .utf8) {
            do {
                var dictToChange = try JSONSerialization.jsonObject(with: data, options: []) as? [String: Any]
                if dictToChange != nil {
                    dictToChange?.merge(dictToAdd, uniquingKeysWith: { (current, _) in current })
                    if let data = try? JSONSerialization.data(withJSONObject: dictToChange!, options: .prettyPrinted) {
                        result = String(data: data, encoding: String.Encoding.utf8)
                    }
                }
            } catch let error {
                Logger.e("Can't add item to json string. Err: \(error)")
            }
        }
        return result
    }
}
