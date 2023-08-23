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
