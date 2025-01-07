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

import { configureLocalization, msg, localized } from "@lit/localize";
import { css, html, LitElement, nothing } from "lit";
import { property } from "lit/decorators.js";
import { sourceLocale, targetLocales } from "./generated/messages";
import { ConnectivityTestRequest, ConnectivityTestResponse, ConnectivityTestResult, OperatingSystem, PlatformMetadata } from "./types";

export * from "./types";

// TODO: only call this once
const Localization = configureLocalization({
  sourceLocale,
  targetLocales,
  loadLocale: (locale: string) => import(`./generated/messages/locales/${locale}.ts`),
});

// TODO: TS says this class doesn't implement ReactiveElement, but it does
// @ts-ignore-next-line
@localized()
export class ConnectivityTestPage extends LitElement {
  @property({ type: Function })
  loadPlatform?: () => Promise<PlatformMetadata>;

  @property({ attribute: false })
  platform?: PlatformMetadata;

  @property({ type: Function })
  onSubmit?: (request: ConnectivityTestRequest) => Promise<ConnectivityTestResponse>;

  @property({ attribute: false })
  isSubmitting = false;

  @property({ attribute: false })
  response?: ConnectivityTestResponse;

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
    const prefix = formData.get("prefix")?.toString();

    if (!accessKey || !domain || !resolvers) {
      return null;
    }

    return {
      accessKey,
      domain,
      resolvers,
      protocols,
      prefix,
    };
  }

  protected async performUpdate() {
    if (!this.platform && this.loadPlatform) {
      this.platform = await this.loadPlatform();
    }

    super.performUpdate();
  }

  async testConnectivity(event: SubmitEvent) {
    event.preventDefault();

    if (!this.formData) {
      return;
    }

    this.isSubmitting = true;

    try {
      this.response = (await this.onSubmit?.(this.formData)) ?? null;
    } catch (error) {
      this.response = new Error(error as string);
    } finally {
      this.isSubmitting = false;
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
      --size-border-switch: 2px;
      --size-corner-radius: 0.2rem;
      --size-gap-inner: 0.25rem;
      --size-gap-narrow: 0.5rem;
      --size-gap: 1rem;
      --size-font-small: 0.75rem;
      --size-font: 1rem;
      --size-font-large: 1.85rem;
      --size-switch: 1.5rem;
      --size-app-width-min: 320px;
      --size-app-width-max: 560px;

      --timing-snappy: 140ms;

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

    .header,
    .header--ios {
      background: var(--color-background-brand);
      display: flex;
      justify-content: center;
      padding: var(--size-gap);
      position: sticky;
      top: 0;
      width: 100%;
    }

    .header--ios {
      padding-top: 3.7rem;
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
      display: flex;
      justify-content: space-around;
    }

    .field-group-item {
      display: flex;
      align-items: center;
      gap: var(--size-gap-narrow);
    }

    .field-label,
    .field-header-label {
      cursor: pointer;
      font-family: var(--font-sans-serif);
      font-size: var(--size-font);
      color: var(--color-text);
    }

    .field-header {
      display: flex;
      align-items: center;
      gap: var(--size-gap-narrow);
      margin-bottom: var(--size-gap-narrow);
    }

    .field-header-label,
    .field-header-label-required {
      font-weight: bold;
    }

    .field-header-label-required {
      color: red;
    }

    .field-header-info {
      opacity: 0.5;
      font-size: var(--size-font-small);
      cursor: help;
    }

    .field-header-info:hover {
      opacity: 1;
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

    .field-input:focus,
    .field-input-textarea:focus {
      border-color: var(--color-background-highlight);
    }

    .field-input-checkbox {
      background: var(--color-background);
      border-radius: var(--size-font);
      border: var(--size-border-switch) solid var(--color-background);
      cursor: pointer;
      display: inline-block;
      flex-shrink: 0;
      height: var(--size-font);
      position: relative;
      transition: background var(--timing-snappy) ease-in-out,
        border var(--timing-snappy) ease-in-out;
      width: var(--size-switch);
    }

    .field-input-checkbox:checked {
      background: var(--color-background-highlight);
      border: var(--size-border-switch) solid var(--color-background-highlight);
    }

    .field-input-checkbox::after {
      background: var(--color-text);
      border-radius: var(--size-font-small);
      color: rgba(0, 0, 0, 0%); /* makes the text invisible */
      content: "🤪"; /* placeholder required for element to show */
      display: inline-block;
      height: var(--size-font-small);
      left: 0;
      position: absolute;
      top: 50%;
      transform: translate(0, -50%);
      transition: transform var(--timing-snappy) ease-in-out,
      left var(--timing-snappy) ease-in-out;
      width: var(--size-font-small);
      will-change: transform, left;
    }

    .field-input-checkbox:checked::after {
      left: 100%;
      transform: translate(-100%, -50%);
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

    .field-input-submit:disabled {
      background: var(--color-background-muted);
      color: var(--color-text-muted);
      cursor: not-allowed;
    }

    .footer {
      background: var(--color-background-muted);
      bottom: 0;
      padding: var(--size-gap);
      position: absolute;
      width: 100%;
    }

    .footer-inner {
      align-items: center;
      display: flex;
      gap: var(--size-gap);
      margin: 0 auto;
      max-width: var(--size-app-width-max);
      padding: 0 var(--size-gap);
    }

    .footer-separator,
    .footer-selector-label {
      color: var(--color-text-muted);
      font-family: var(--font-sans-serif);
      font-size: var(--size-font-small);
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

    @media (prefers-reduced-motion: reduce) {
      .field-input-checkbox,
      .field-input-checkbox::after {
        transition: none;
      }
    }

    @media (prefers-contrast: more) {
      :host {
        --size-border: 2px;
      }

      .field-group {
        border: var(--size-border) solid var(--color-text);
        background: var(--color-background);
      }

      .field-header-info {
        opacity: 1;
      }

      .field-input-checkbox {
        border-color: var(--color-text);
      }

      .field-input-checkbox:checked {
        background: var(--color-background);
        border-color: var(--color-text);
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

    
    @media only screen and (max-width: 480px) {
      /* if mobile */
      .results-list-item-data {
        display: flex;
        flex-direction: column;
        gap: var(--size-gap-inner);
      }

      .footer {
        display: none;
      }

      .field-header-info {
        display: none;
      }
    }
  `;

  render() {
    // TODO: move language definitions to a centralized place
    return html`<main dir="${this.locale === "fa-IR" ? "rtl" : "ltr"}">
      <header class=${this.platform?.operatingSystem === OperatingSystem.IOS ? "header--ios" : "header"}>
        <h1 class="header-text">${msg("Outline Connectivity Test")}</h1>
      </header>
      ${this.renderResults()}
      <form class="form" @submit=${this.testConnectivity}>
        <fieldset class="field">
          <span class="field-header">
            <label class="field-header-label" for="accessKey">
              ${msg("Outline Access Key")}
            </label>
            <span class="field-header-label-required">*</span>
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
              ${msg("DNS Resolvers to Try")}
            </label>
            <span class="field-header-label-required">*</span>
            <i
              class="field-header-info"
              title=${msg(
                "A DNS resolver is an online service that returns the direct IP address of a given website domain."
              )}
              >ℹ️</i
            >
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
            <span class="field-header-label-required">*</span>
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
          <label class="field-header-label"
            >${msg("Protocols to Check")}
          </label>
          <i
            class="field-header-info"
            title=${msg(
              "The main difference between TCP (transmission control protocol) and UDP (user datagram protocol) is that TCP is a connection-based protocol and UDP is connectionless. While TCP is more reliable, it transfers data more slowly. UDP is less reliable but works more quickly."
            )}
            >ℹ️</i
          >
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
            <label class="field-label" for="tcp">TCP</label>
          </span>

          <span class="field-group-item">
            <input
              class="field-input-checkbox"
              type="checkbox"
              name="udp"
              id="udp"
              checked
            />
            <label class="field-label" for="udp">UDP</label>
          </span>
        </fieldset>

        <fieldset class="field">
          <span class="field-header">
            <label class="field-header-label" for="prefix">
              ${msg("TCP Stream Prefix")}
            </label>
            <i
              class="field-header-info"
              title=${msg(
                "The TCP stream prefix is a plaintext string appended to the start of the encrypted TCP payload, making the data transfer appear like an acceptable method."
              )}
              >ℹ️</i
            >
          </span>
          <select class="field-input" name="prefix" id="prefix">
            <option value="">${msg("None")}</option>
            <option value="POST ">POST</option>
            <option value="HTTP/1.1 ">HTTP/1.1</option>
          </select>
        </fieldset>

        <input
          ?disabled=${this.isSubmitting}
          class="field-input-submit"
          type="submit"
          value="${this.isSubmitting ? msg("Testing...") : msg("Run Test")}"
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
            <option dir="rtl" value="fa-IR">فارسی</option>
          </select>

          <i class="footer-separator">•</i>

          <label class="footer-selector-label" for="theme"
            >${msg("Theme")}</label
          >

          <!--
              Only serves to currently communicate to the user that
              the theme should respond to the system settings.
            -->
          <select class="footer-selector" id="theme">
            <option value="system">${msg("System")}</option>
          </select>
        </div>
      </footer>
    </main>`;
  }

  renderResults() {
    if (!this.response) {
      return nothing;
    }

    return html`<dialog open class="results">
      <header class="results-header">
        <h2 class="results-header-text">${msg("Test Results")}</h2>
        <button
          class="results-header-close"
          @click=${() => (this.response = null)}
        >
          ✕
        </button>
      </header>
      ${this.renderResultsList(this.response)}
    </dialog>`;
  }

  renderResultsList(response: ConnectivityTestResult[] | Error) {
    if (response instanceof Error) {
      return html`
        <ul class="results-list">
          <li class="results-list-item results-list-item--failure">
            <i class="results-list-item-status">✖</i>
            <dl class="results-list-item-data">
              <dt class="results-list-item-data-key">${msg("Error")}</dt>
              <dd class="results-list-item-data-value">${response.message}</dd>
            </dl>
          </li>
        </ul>
      `;
    }

    return html` <ul class="results-list">
      ${response.map((result) => {
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

            <dt class="results-list-item-data-key">${msg("Resolver")}</dt>
            <dd class="results-list-item-data-value">${result.resolver}</dd>

            <dt class="results-list-item-data-key">${msg("Time")}</dt>
            <dd class="results-list-item-data-value">${result.durationMs}ms</dd>
          </dl>
        </li>`;
      })}
    </ul>`;
  }
}
