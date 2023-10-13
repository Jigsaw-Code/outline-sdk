import { html } from "lit";
import { msg } from "@lit/localize";

import * as Styles from "./styles.css";

const DEFAULT_DNS_RESOLVERS = `
8.8.8.8
2001:4860:4860::8888`;

export const Form = (connectivityTestHandler: (event: SubmitEvent) => Promise<void>, isSubmitting: boolean) => html`
  <form class="${Styles}" @submit=${connectivityTestHandler}>
    <fieldset class="${Styles.Field}">
      <span class="${Styles.FieldHeader}">
        <label for="accessKey" class="${Styles.FieldHeaderLabel}">
          ${msg("Outline Access Key")}
        </label>
        <span class="${Styles.FieldHeaderRequired}">
          *
        </span>
      </span>
      <textarea
        class="${Styles.FieldTextInput}"
        name="accessKey"
        id="accessKey"
        required
      ></textarea>
    </fieldset>

    <fieldset class="${Styles.Field}">
      <span class="${Styles.FieldHeader}">
        <label for="resolvers" class="${Styles.FieldHeaderLabel}">
          ${msg("DNS Resolvers to Try")}
        </label>
        <span class="${Styles.FieldHeaderRequired}">
          *
        </span>
        <i class="${Styles.FieldHeaderInformation}"
          title=${msg(
  "A DNS resolver is an online service that returns the direct IP address of a given website domain."
  )}>
        ℹ️
        </i>
      </span>
      <textarea
        class="${Styles.FieldTextInput}"
        name="resolvers"
        id="resolvers"
        required
      >${DEFAULT_DNS_RESOLVERS}</textarea>
    </fieldset>

    <fieldset class="${Styles.Field}">
      <span class="${Styles.FieldHeader}">
        <label for="domain"
          class="${Styles.FieldHeaderLabel}">
          ${msg("Domain to Test")}
        </label>
        <span class="${Styles.FieldHeaderRequired}">
          *
        </span>
      </span>
      <input
        class="${Styles.FieldTextInput}"
        name="domain"
        id="domain"
        required
        value="example.com"
      />
    </fieldset>

    <span class="${Styles.FieldHeader}">
      <label class="${Styles.FieldHeaderLabel}"
        >${msg("Protocols to Check")}
      </label>
      <i
        class="${Styles.FieldHeaderInformation}"
        title=${msg(
  "The main difference between TCP (transmission control protocol) and UDP (user datagram protocol) is that TCP is a connection-based protocol and UDP is connectionless. While TCP is more reliable, it transfers data more slowly. UDP is less reliable but works more quickly."
  )}
        >ℹ️</i
      >
    </span>
    <fieldset class="${Styles.FieldGroup}">
      <span class="${Styles.FieldGroupItem}">
        <input
          class="${Styles.FieldCheckbox}"
          type="checkbox"
          name="tcp"
          id="tcp"
          checked
        />
        <label class="${Styles.FieldLabel}" for="tcp">TCP</label>
      </span>

      <span class="${Styles.FieldGroupItem}">
        <input
          class="${Styles.FieldCheckbox}"
          type="checkbox"
          name="udp"
          id="udp"
          checked
        />
        <label class="${Styles.FieldLabel}" for="udp">UDP</label>
      </span>
    </fieldset>

    <fieldset class="${Styles.Field}">
      <span class="${Styles.FieldHeader}">
        <label for="prefix" 
          class="${Styles.FieldHeaderLabel}">
          ${msg("TCP Stream Prefix")}
        </label>
        <i
          class="${Styles.FieldHeaderInformation}"
          title=${msg(
  "The TCP stream prefix is a plaintext string appended to the start of the encrypted TCP payload, making the data transfer appear like an acceptable method."
  )}
          >ℹ️</i
        >
      </span>
      <select class="${Styles.FieldInput}" name="prefix" id="prefix">
        <option value="">${msg("None")}</option>
        <option value="POST ">POST</option>
        <option value="HTTP/1.1 ">HTTP/1.1</option>
      </select>
    </fieldset>

    <input
      ?disabled=${isSubmitting}
      class="${Styles.Submit}"
      type="submit"
      value="${isSubmitting ? msg("Testing...") : msg("Run Test")}"
    />
  </form>
`;
