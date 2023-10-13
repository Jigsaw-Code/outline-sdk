import { html } from "lit";
import { msg } from "@lit/localize";

import * as Styles from "./styles.css";

export const Footer = (changeHandler: (event: { target: HTMLSelectElement }) => void) => html`
  <footer class="${Styles.Main}">
    <div class="${Styles.Inner}">
      <label for="language" class="${Styles.Separator}">
        ${msg("Language")}
      </label>
      <select id="language"
        class="${Styles.Selector}"
        @change=${changeHandler}
      >
        <option value="en">English</option>
        <option value="es-419">Español</option>
        <option value="zh-Hans">中文</option>
        <option dir="rtl" value="fa-IR">فارسی</option>
      </select>

      <i class="${Styles.Separator}">•</i>

      <label for="theme" class="${Styles.Separator}">
        ${msg("Theme")}
      </label>

      <!--
          Only serves to currently communicate to the user that
          the theme should respond to the system settings.
        -->
      <select id="theme" class="${Styles.Selector}">
        <option value="system">${msg("System")}</option>
      </select>
    </div>
  </footer>`