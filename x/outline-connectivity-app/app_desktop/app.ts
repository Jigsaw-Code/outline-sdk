// backend
import * as SharedBackend from "shared_backend";
import * as DesktopBackend from "./generated/wailsjs/go/main/App";

// frontend
import * as SharedFrontend from "shared_frontend";
import { LitElement, html } from "lit";
import { customElement } from "lit/decorators.js";

SharedFrontend.registerAllElements();

// main
@customElement("app-main")
export class AppMain extends LitElement {
  backend = SharedBackend.from(DesktopBackend);

  render() {
    return html`<connectivity-test-page .onSubmit=${this.backend.connectivityTest} />`;
  }
}
