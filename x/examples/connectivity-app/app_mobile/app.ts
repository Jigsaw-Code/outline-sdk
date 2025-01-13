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

// backend
import { registerPlugin } from "@capacitor/core";

// Capacitor requires passing in a root object to each native call,
// that is then accessed via APIs like `call.getString("objectKey")`.
const MobileBackend = registerPlugin<{
  Request(request: { resourceName: string; parameters: string }): Promise<{ error: string; body: string }>;
}>("MobileBackend");

async function requestBackend<T, K>(resourceName: string, parameters: T): Promise<K> {
  const response = await MobileBackend.Request({
    resourceName, parameters: JSON.stringify(parameters)
  });

  if (response.error) {
    throw new Error(response.error);
  }

  return JSON.parse(response.body);
}


// frontend
import { LitElement, html } from "lit";
import type { ConnectivityTestRequest, ConnectivityTestResponse } from "shared_frontend";
import { customElement } from "lit/decorators.js";
import * as SharedFrontend from "shared_frontend";

SharedFrontend.registerAllElements();

// main
@customElement("app-main")
export class AppMain extends LitElement {
  render() {
    return html`<connectivity-test-page 
    .loadPlatform=${() => requestBackend<void, SharedFrontend.PlatformMetadata>("Platform", void 0)}
    .onSubmit=${(parameters: ConnectivityTestRequest) =>
        requestBackend<ConnectivityTestRequest, ConnectivityTestResponse>(
          "ConnectivityTest", parameters
        )
      } />`;
  }
}
