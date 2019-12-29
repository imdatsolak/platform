//
//  ServerRequest.swift
//  JobServer
//

import Foundation
import NIO
import NIOHTTP1

public class ServerRequest {

    public let header: HTTPRequestHead // <= from NIOHTTP1
    public var userInfo = [ String: Any]()
    public var jsonData: Data? = nil {
        didSet {
            self.jsonRead()
        }
    }

    init(header: HTTPRequestHead) {
        self.header = header
    }

    func jsonRead() {
        if let data = self.jsonData {
            let jsonObj = try? JSONSerialization.jsonObject(with: data, options: .allowFragments)
            if let json = jsonObj {
                Logger.d("Request JSON:\n" + String(describing: json) + "\n")
            }
        }
    }

    func mapJsonTo<T: Decodable>(type: T.Type) -> T? {
        if let data = jsonData {

            if let decodedObj: T = try? JSONDecoder().decode(type, from: data) {
                return decodedObj
            }
                else {
                     Logger.e("Json decode error")
            }
        }
        return nil
    }
}
