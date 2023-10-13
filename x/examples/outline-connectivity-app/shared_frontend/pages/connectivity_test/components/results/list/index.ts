import { html, nothing } from "lit";
import { msg } from "@lit/localize";

import * as Styles from "./styles.css";
import { ConnectivityTestResult } from "../../../types";

export const List = (response?: ConnectivityTestResult[]) => {
  if (!response) return nothing;

  return html`
    <ul class="${Styles.Main}">
      ${response.map((result) => {
        const isSuccess = !result.error;
  
        return html`
          <li class="${isSuccess ? Styles.SuccessItem : Styles.FailureItem}">
            <i class="${Styles.ItemStatus}">
              ${isSuccess ? "✔" : "✖"}
            </i>
            <dl class="${Styles.ItemData}">
              <dt class="${Styles.ItemDataKey}">${msg("Protocol")}</dt>
              <dd class="${Styles.ItemDataValue}">
                ${result.proto.toUpperCase()}
              </dd>
  
              <dt class="${Styles.ItemDataKey}">${msg("Resolver")}</dt>
              <dd class="${Styles.ItemDataValue}">${result.resolver}</dd>
  
              <dt class="${Styles.ItemDataKey}">${msg("Time")}</dt>
              <dd class="${Styles.ItemDataValue}">${result.durationMs}ms</dd>
            </dl>
          </li>`;
      })}
    </ul>
  `;
}