// Copyright 2023 The Outline Authors
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

struct FrontendRequest: Codable {
    var resourceName: String
    var parameters: String
}

struct FrontendResponse: Decodable {
    var body: String
    var error: String
}

@objc(MobileBackendPlugin)
public class MobileBackendPlugin: CAPPlugin {
    @objc func Request(_ call: CAPPluginCall) {
        let response: FrontendResponse

        let encoder = JSONEncoder()
        let decoder = JSONDecoder()

        do {
            let rawRequest = try encoder.encode(
                FrontendRequest(
                    resourceName: call.getString("resourceName")!,
                    parameters: call.getString("parameters") ?? "{}"
                )
            )
            
            response = try decoder.decode(
                FrontendResponse.self,
                // TODO: make this non blocking: https://stackoverflow.com/a/69381330
                from: Shared_backendHandleRequest(rawRequest)!
            )
        } catch {
            return call.resolve([
                "error": error
            ])
        }
            
        return call.resolve([
            "body": response.body,
            "error": response.error
        ])
    }
}
