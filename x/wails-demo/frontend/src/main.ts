import { configureLocalization, msg, localized } from "@lit/localize";
import { css, html, LitElement } from "lit";
import { customElement, property } from "lit/decorators.js";
import { main } from "../wailsjs/go/models";
import { sourceLocale, targetLocales } from "./generated/locale-codes.js";

import * as wailsApp from "../wailsjs/go/main/App";

const Localization = configureLocalization({
  sourceLocale,
  targetLocales,
  loadLocale: (locale: string) => import(`/src/generated/locales/${locale}.ts`),
});

@customElement("main-page")
@localized()
export class MainPage extends LitElement {
  get locale() {
    return Localization.getLocale();
  }

  set locale(newLocale: string) {
    Localization.setLocale(newLocale);
  }

  get formData() {
    const formElement = this.shadowRoot?.querySelector("form");

    if (!formElement) {
      return null;
    }

    const formData = new FormData(formElement);

    const accessKey = formData.get("accessKey")?.toString().trim();
    const domain = formData.get("domain")?.toString().trim();
    const resolvers =
      formData
        .get("resolvers")
        ?.toString()
        .split(/,?\s+/)
        .map((line) => line.trim()) || null;
    const protocols = {
      tcp: formData.get("tcp") === "on",
      udp: formData.get("udp") === "on",
    };

    if (!accessKey || !domain || !resolvers) {
      return null;
    }

    return {
      accessKey,
      domain,
      resolvers,
      protocols,
    };
  }

  @property({ type: Array })
  testResults: main.ConnectivityTestResult[] | Error | null = null;

  async testConnectivity(event: SubmitEvent) {
    event.preventDefault();

    if (!this.formData) {
      return;
    }

    try {
      this.testResults = await wailsApp.TestConnectivity(
        this.formData.accessKey,
        this.formData.domain,
        this.formData.resolvers,
        this.formData.protocols
      );
    } catch (error) {
      this.testResults = new Error(error as string);
    }
  }

  static styles = css`
    * {
      all: initial;
      box-sizing: border-box;
    }

    :host {
      --font-sans-serif: "Jigsaw Sans", "Helvetica Neue", Helvetica, Arial,
        sans-serif;
      --font-monospace: "Menlo", Courier, monospace;

      --size-border: 0.5px;
      --size-corner-radius: 0.2rem;
      --size-gap-inner: 0.25rem;
      --size-gap-narrow: 0.5rem;
      --size-gap: 1rem;
      --size-font-small: 0.75rem;
      --size-font: 1rem;
      --size-font-large: 1.85rem;
      --size-app-width-min: 560px;
      --size-app-width-max: 560px;

      --jigsaw-black: hsl(300, 3%, 7%);
      --jigsaw-white: white;
      --jigsaw-green: hsl(153, 39%, 15%);
      --jigsaw-green-medium: hsl(156, 33%, 31%);
      --jigsaw-green-light: hsl(159, 13%, 74%);
      --jigsaw-gray: hsl(0, 0%, 73%);
      --jigsaw-gray-medium: hsl(0, 0%, 90%);
      --jigsaw-gray-light: hsl(0, 0%, 98%);

      --color-text: var(--jigsaw-black);
      --color-text-brand: var(--jigsaw-white);
      --color-text-highlight: var(--jigsaw-white);
      --color-text-muted: var(--jigsaw-gray);
      --color-background: var(--jigsaw-white);
      --color-background-brand: var(--jigsaw-green);
      --color-background-highlight: hsl(156, 33%, 55%);
      --color-background-muted: var(--jigsaw-gray-medium);
      --color-success-text: var(--jigsaw-green-medium);
      --color-success-background: var(--jigsaw-green-light);
      --color-error-text: hsl(0, 33%, 31%);
      --color-error-background: hsl(0, 13%, 74%);
    }

    @media (prefers-color-scheme: dark) {
      :host {
        --color-text: var(--jigsaw-gray-medium);
        --color-text-brand: var(--jigsaw-white);
        --color-text-highlight: var(--jigsaw-white);
        --color-text-muted: hsl(0, 0%, 31%);
        --color-background: var(--jigsaw-black);
        --color-background-brand: var(--jigsaw-green);
        --color-background-highlight: var(--jigsaw-green-medium);
        --color-background-muted: hsl(0, 0%, 15%);
        --color-success-text: var(--jigsaw-white);
        --color-success-background: hsl(156, 50%, 31%);
        --color-error-text: var(--jigsaw-white);
        --color-error-background: hsl(0, 50%, 45%);
      }
    }

    main {
      background: var(--color-background);
      display: block;
      min-width: var(--size-app-width-min);
      min-height: 100vh;
      position: relative;
      width: 100vw;
    }

    .header {
      background: var(--color-background-brand);
      display: flex;
      justify-content: center;
      padding: var(--size-gap);
      position: sticky;
      top: 0;
      width: 100%;
    }

    .header-text {
      color: var(--color-text-brand);
      font-family: var(--font-sans-serif);
      font-size: var(--size-font-large);
      max-width: var(--size-app-width-max);
      text-align: center;
    }

    .results {
      background: var(--color-background-muted);
      border-radius: var(--size-corner-radius);
      display: block;
      margin: var(--size-gap) auto;
      margin-bottom: calc(var(--size-gap) * 2);
      max-width: var(--size-app-width-max);
      width: 100%;
    }

    .results-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: var(--size-gap-narrow) var(--size-gap);
    }

    .results-header-text {
      color: var(--color-text);
      font-family: var(--font-sans-serif);
      font-size: var(--size-font);
    }

    .results-header-close {
      color: var(--color-text);
      cursor: pointer;
      font-family: var(--font-sans-serif);
      font-size: var(--size-font);
      opacity: 0.5;
    }

    .results-header-close:hover {
      opacity: 1;
    }

    .results-list {
      list-style: none;
    }

    .results-list-item {
      align-items: center;
      display: flex;
      gap: var(--size-gap);
      padding: var(--size-gap);
      border-style: solid;
      border-width: 0 0 var(--size-border) 0;
    }

    .results-list-item--success {
      background: var(--color-success-background);
      border-color: var(--color-success-text);
    }

    .results-list-item--failure {
      background: var(--color-error-background);
      border-color: var(--color-error-text);
    }

    .results-list-item:last-child {
      border-bottom-left-radius: var(--size-corner-radius);
      border-bottom-right-radius: var(--size-corner-radius);
      border-bottom: none;
    }

    .results-list-item--success * {
      color: var(--color-success-text);
    }

    .results-list-item--failure * {
      color: var(--color-error-text);
    }

    .results-list-item-status {
      font-family: var(--font-sans-serif);
      font-size: var(--size-font);
      flex-shrink: 0;
    }

    .results-list-item-data {
      display: grid;
      flex-grow: 1;
      grid-auto-flow: column;
      grid-auto-columns: 1fr;
      grid-template-rows: repeat(2, min-content);
      row-gap: var(--size-narrow-gap);
      column-gap: var(--size-gap-narrow);
    }

    .results-list-item-data-key {
      opacity: 0.5;
      font-family: var(--font-sans-serif);
      font-size: var(--size-font-small);
      text-transform: uppercase;
      grid-row-start: 1;
    }

    .results-list-item-data-value {
      font-family: var(--font-monospace);
      font-size: var(--size-font);
    }

    .form {
      display: block;
      gap: var(--size-gap-narrow);
      margin: 0 auto;
      max-width: var(--size-app-width-max);
      padding: var(--size-gap);
      padding-bottom: calc(var(--size-gap) * 6);
      margin-top: var(--size-gap);
    }

    .field {
      display: block;
      margin-bottom: var(--size-gap);
    }

    .field-group {
      background: var(--color-background-muted);
      border-radius: var(--size-corner-radius);
      padding: var(--size-gap);
    }

    .field-group-item {
      display: flex;
      align-items: center;
      gap: var(--size-gap-narrow);
      margin-bottom: var(--size-gap-inner);
    }

    .field-label,
    .field-header-label {
      cursor: pointer;
      font-family: var(--font-sans-serif);
      font-size: var(--size-font);
      color: var(--color-text);
    }

    .field-header {
      display: block;
      margin-bottom: var(--size-gap-narrow);
    }

    .field-header-label,
    .field-header-label-required {
      font-weight: bold;
    }

    .field-header-label-required {
      color: red;
    }

    .field-input,
    .field-input-textarea {
      border-radius: var(--size-corner-radius);
      border: var(--size-border) solid var(--color-text);
      display: block;
      font-family: var(--font-monospace);
      padding: var(--size-gap-narrow);
      width: 100%;
      background: var(--color-background-muted);
      color: var(--color-text);
    }

    .field-input-checkbox {
      display: inline-block;
      flex-shrink: 0;
      width: var(--size-font);
      height: var(--size-font);
      cursor: pointer;
      background: var(--color-background);
      border: var(--size-border) solid var(--color-text);
      position: relative;
    }

    .field-input:focus,
    .field-input-textarea:focus,
    .field-input-checkbox:focus,
    .field-input-checkbox:checked {
      border-color: var(--color-background-highlight);
    }

    .field-input-checkbox:checked {
      background: var(--color-background-highlight);
    }

    .field-input-checkbox:checked::after {
      align-items: center;
      color: var(--color-text-highlight);
      content: "✔";
      display: flex;
      font-size: var(--size-font-small);
      justify-content: center;
      left: 0;
      position: absolute;
      line-height: var(--size-font);
      top: 0;
      width: 100%;
    }

    .field-input-submit {
      background: var(--color-text);
      border-radius: var(--size-corner-radius);
      color: var(--color-background);
      cursor: pointer;
      font-family: var(--font-sans-serif);
      padding: var(--size-gap-narrow);
      text-align: center;
    }

    .field-input-submit:focus,
    .field-input-submit:hover {
      background: var(--color-background-highlight);
      color: var(--color-text-highlight);
    }

    .footer {
      position: absolute;
      bottom: 0;
      padding: var(--size-gap);
      background: var(--color-background-muted);
      width: 100%;
    }

    .footer-inner {
      display: block;
      padding: 0 var(--size-gap);
      max-width: var(--size-app-width-max);
      margin: 0 auto;
    }

    .footer-separator {
      color: var(--color-text-muted);
      font-family: var(--font-sans-serif);
      font-size: var(--size-font-small);
      margin: 0 var(--size-gap);
    }

    .footer-selector-label {
      color: var(--color-text-muted);
      font-family: var(--font-sans-serif);
      font-size: var(--size-font-small);
      margin-right: var(--size-gap-inner);
    }

    .footer-selector {
      background: var(--color-text-muted);
      border-radius: var(--size-corner-radius);
      color: var(--color-background-muted);
      cursor: pointer;
      font-family: var(--font-sans-serif);
      text-align-last: center;
      font-size: var(--size-font-small);
      padding: var(--size-gap-inner) var(--size-gap-narrow);
    }

    .footer-selector:focus,
    .footer-selector:hover {
      background: var(--color-background-highlight);
      color: var(--color-text-highlight);
    }

    @media (prefers-contrast: more) {
      :host {
        --size-border: 2px;
      }

      .field-group {
        border: var(--size-border) solid var(--color-text);
        background: var(--color-background);
      }

      .results-header {
        border: var(--size-border) solid var(--color-text);
        background: var(--color-background);
      }

      .results-header-close {
        opacity: 1;
      }

      .results-list-item,
      .results-list-item:last-child {
        background: var(--color-background);
        border-width: 0 var(--size-border) var(--size-border) var(--size-border);
        border-bottom: var(--size-border) solid var(--color-text);
      }

      .results-list-item-data-key,
      .results-list-item-data-value {
        opacity: 1;
        color: var(--color-text);
      }

      .field-input,
      .field-input-textarea {
        background: var(--color-background);
      }

      .footer {
        border-top: var(--size-border) solid var(--color-text);
        background: var(--color-background);
      }

      .footer-separator {
        color: var(--color-text);
      }

      .footer-selector-label {
        color: var(--color-text);
      }

      .footer-selector {
        background: var(--color-text);
        color: var(--color-background);
      }
    }
  `;

  render() {
    return html`<main>
      <header class="header">
        <h1 class="header-text">${msg("Outline Connectivity Test")}</h1>
      </header>
      ${this.renderResults()}
      <form class="form" @submit=${this.testConnectivity}>
        <fieldset class="field">
          <span class="field-header">
            <label class="field-header-label" for="accessKey">
              ${msg("Outline Access Key")}
            </label>
            <span class="field-header-label-required"> * </span>
          </span>
          <textarea
            class="field-input-textarea"
            name="accessKey"
            id="accessKey"
            required
          ></textarea>
        </fieldset>

        <fieldset class="field">
          <span class="field-header">
            <label class="field-header-label" for="resolvers">
              ${msg("Domain Name Servers to Use")}
            </label>
            <span class="field-header-label-required">*</span>
          </span>
          <textarea
            class="field-input-textarea"
            name="resolvers"
            id="resolvers"
            required
          >
8.8.8.8
2001:4860:4860::8888</textarea
          >
        </fieldset>

        <fieldset class="field">
          <span class="field-header">
            <label class="field-header-label" for="domain"
              >${msg("Domain to Test")}</label
            >
            <span class="field-header-label-required"> * </span>
          </span>
          <input
            class="field-input"
            name="domain"
            id="domain"
            required
            value="example.com"
          />
        </fieldset>

        <span class="field-header">
          <label class="field-header-label">${msg("Protocols to Check")}</label>
        </span>
        <fieldset class="field field-group">
          <span class="field-group-item">
            <input
              class="field-input-checkbox"
              type="checkbox"
              name="tcp"
              id="tcp"
              checked
            />
            <label class="field-label" for="tcp"
              >${msg("Transmission Control Protocol (TCP)")}</label
            >
          </span>

          <span class="field-group-item">
            <input
              class="field-input-checkbox"
              type="checkbox"
              name="udp"
              id="udp"
              checked
            />
            <label class="field-label" for="udp"
              >${msg("User Datagram Protocol (UDP)")}</label
            >
          </span>
        </fieldset>

        <input
          class="field-input-submit"
          type="submit"
          value="${msg("Run Test")}"
        />
      </form>
      <footer class="footer">
        <div class="footer-inner">
          <label class="footer-selector-label" for="language"
            >${msg("Language")}</label
          >
          <select
            class="footer-selector"
            id="language"
            @change=${({ target }: { target: HTMLSelectElement }) =>
              (this.locale = target.value)}
          >
            <option value="en">English</option>
            <option value="es-419">Español</option>
            <option value="zh-Hans">中文</option>
          </select>

          <i class="footer-separator">•</i>

          <label class="footer-selector-label" for="theme"
            >${msg("Theme")}</label
          >
          <select class="footer-selector" id="theme">
            <option value="system">${msg("System")}</option>
            <!-- <option value="light">${msg("Light")}</option>
            <option value="dark">${msg("Dark")}</option>
            <option value="high-contrast">${msg("High Contrast")}</option> -->
          </select>
        </div>
      </footer>
    </main>`;
  }

  renderResults() {
    if (!this.testResults) {
      return;
    }

    return html`<section class="results">
      <header class="results-header">
        <h2 class="results-header-text">${msg("Test Results")}</h2>
        <button
          class="results-header-close"
          @click=${() => (this.testResults = null)}
        >
          ✕
        </button>
      </header>
      ${this.renderResultsList(this.testResults)}
    </section>`;
  }

  renderResultsList(results: main.ConnectivityTestResult[] | Error) {
    if (results instanceof Error) {
      return html`
        <ul class="results-list">
          <li class="results-list-item results-list-item--failure">
            <i class="results-list-item-status">✖</i>
            <dl class="results-list-item-data">
              <dt class="results-list-item-data-key">${msg("Error")}</dt>
              <dd class="results-list-item-data-value">${results.message}</dd>
            </dl>
          </li>
        </ul>
      `;
    }

    return html` <ul class="results-list">
      ${results.map((result) => {
        const isSuccess = !result.error;

        return html`<li
          class="results-list-item ${isSuccess
            ? "results-list-item--success"
            : "results-list-item--failure"}"
        >
          <i class="results-list-item-status">${isSuccess ? "✔" : "✖"}</i>
          <dl class="results-list-item-data">
            <dt class="results-list-item-data-key">${msg("Protocol")}</dt>
            <dd class="results-list-item-data-value">
              ${result.proto.toUpperCase()}
            </dd>

            <!-- <dt class="results-list-item-data-key">${msg("Prefix")}</dt>
              <dd class="results-list-item-data-value">
                ${result.prefix || msg("NONE")}
              </dd> -->

            <dt class="results-list-item-data-key">${msg("Resolver")}</dt>
            <dd class="results-list-item-data-value">${result.resolver}</dd>

            <dt class="results-list-item-data-key">${msg("Ping")}</dt>
            <dd class="results-list-item-data-value">
              ${result.durationMs}ms
            </dd>
          </dl>
        </li>`;
      })}
    </ul>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "main-page": MainPage;
  }
}
