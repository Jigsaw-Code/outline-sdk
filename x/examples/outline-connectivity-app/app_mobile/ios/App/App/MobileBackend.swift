// Copyright 2023 Jigsaw Operations LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
