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

// backend
import { registerPlugin } from "@capacitor/core";
import * as SharedBackend from "shared_backend";

const MobileBackend = registerPlugin<SharedBackend.Invokable>("MobileBackend");

// frontend
import { LitElement, html } from "lit";
import { customElement } from "lit/decorators.js";
import * as SharedFrontend from "shared_frontend";

SharedFrontend.registerAllElements();

// main
@customElement("app-main")
export class AppMain extends LitElement {
  backend = SharedBackend.from(MobileBackend)

  render() {
    return html`<connectivity-test-page .onSubmit=${this.backend.connectivityTest} />`;
  }
}
