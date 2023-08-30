//
//  Library.swift
//  App
//
//  Created by Daniel LaCosse on 8/7/23.
//

import Foundation
import Capacitor
import SharedBackend

struct RawBackendCallInputMessage: Codable {
    var method: String
    var input: String
}

struct RawBackendCallOutputMessage: Decodable {
    var result: String
    var errors: [String]
}

@objc(BackendPlugin)
public class BackendPlugin: CAPPlugin {
    @objc func Invoke(_ call: CAPPluginCall) {
        let outputMessage: RawBackendCallOutputMessage

        let encoder = JSONEncoder()
        let decoder = JSONDecoder()

        do {
            let rawInputMessage = try encoder.encode(
                RawBackendCallInputMessage(
                    method: call.getString("method")!,
                    input: call.getString("input")!
                )
            )
            
            outputMessage = try decoder.decode(
                RawBackendCallOutputMessage.self,
                from: Shared_backendSendRawCall(rawInputMessage)!
            )
        } catch {
            return call.resolve([
                "errors": [error]
            ])
        }
            
        return call.resolve([
            "result": outputMessage.result,
            "errors": outputMessage.errors
        ])
    }
}
