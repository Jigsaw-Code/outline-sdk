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

import { configureLocalization, msg, localized } from "@lit/localize";
import { css, html, LitElement } from "lit";
import { property } from "lit/decorators.js";
import { sourceLocale, targetLocales } from "./generated/messages";
import * as Styles from "./styles.css";
import * as Components from "./components";

import type { ConnectivityTestRequest, ConnectivityTestResponse, ConnectivityTestResult } from "./types";

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
  onSubmit?: (request: ConnectivityTestRequest) => Promise<ConnectivityTestResponse>;

  @property({ attribute: false })
  isSubmitting = false;

  @property({ attribute: false })
  testResponse: ConnectivityTestResponse = null;

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

  async testConnectivity(event: SubmitEvent) {
    event.preventDefault();

    if (!this.formData) {
      return;
    }

    this.isSubmitting = true;

    try {
      this.testResponse = (await this.onSubmit?.(this.formData)) ?? null;
    } catch (error) {
      this.testResponse = new Error(error as string);
    } finally {
      this.isSubmitting = false;
    }
  }

  static styles = css`
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

    @media (prefers-contrast: more) {
      :host {
        --size-border: 2px;
      }
    }

      .results-list-item-data-key,
      .results-list-item-data-value {
        opacity: 1;
        color: var(--color-text);
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

      /* if ios */
      .header {
        padding-top: 3.7rem;
      }
    }
  `;

  render() {
    // TODO: move language definitions to a centralized place
    return html`<main class=${Styles.Main} dir="${this.locale === "fa-IR" ? "rtl" : "ltr"}">
      ${Components.Header()}
      ${Components.Results()}
      ${Components.Form(this.testConnectivity, this.isSubmitting)}
      ${Components.Footer(({ target }: { target: HTMLSelectElement }) => (this.locale = target.value))}
    </main>`;
  }
}
