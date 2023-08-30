import "./theme.css";
import { ConnectivityTestPage } from "./pages";

export function registerAllElements() {
  window.customElements.define("connectivity-test-page",ConnectivityTestPage);
}