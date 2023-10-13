import { html } from "lit";
import { msg } from "@lit/localize";
import * as Styles from "./styles.css";

export const Header = () => html`
  <header class="${Styles.Main}">
    <h1 class="${Styles.Text}">
      ${msg("Outline Connectivity Test")}
    </h1>
  </header>
`;