import { html, nothing } from "lit";
import { msg } from "@lit/localize";

import { style } from "@vanilla-extract/css";
import { App } from "../../../../assets/themes";

import * as Styles from "./styles.css";

import { ConnectivityTestResponse } from "../../types";
import { List } from "./list";

export const Results = (cancelHandler: () => void, response?: ConnectivityTestResponse[]) => {
  if (!response) return nothing;

  return html`<dialog open class="${style({
    background: App.Theme.color.backgroundMuted,
    borderRadius: App.Theme.size.cornerRadius,
    display: "block",
    margin: App.Theme.size.gap,
    marginBottom: `calc(${App.Theme.size.gap} * 2)`,
    maxWidth: App.Theme.size.appWidthMax,
    width: "100%",
  })}">
    <header class="${Styles.Header}">
      <h2 class="${Styles.HeaderText}">${msg("Test Results")}</h2>
      <button
        class="${Styles.HeaderClose}"
        @click=${cancelHandler}
      >
        ✕
      </button>
    </header>
    ${List(response)}
  </dialog>`
}
