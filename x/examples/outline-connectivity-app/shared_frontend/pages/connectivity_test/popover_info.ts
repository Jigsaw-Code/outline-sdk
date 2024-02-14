import { css, html, LitElement, nothing } from "lit";
import { property, customElement, query } from "lit/decorators.js";
import { unsafeHTML } from 'lit/directives/unsafe-html.js';

@customElement('info-popup')
export class InfoPopup extends LitElement {
  @property({ type: Boolean }) isPopupVisible = false;
  @property({ type: String }) popupText = 'This is an <strong>info</strong> pop-up. Click the button to toggle visibility.'

  @query('#my-popover')
  popoverElement: HTMLElement;

  static styles = css`
  .popup {
    top: -50%; /* Position above the element */
    font-family: var(--font-sans-serif);
    padding: 25px;
    max-width: 75%;
    box-sizing: border-box;
    border: 1px solid black;
    background-color: white;
    box-shadow: 0px 0px 10px rgba(0,0,0,0.7);
  }
  .close-button {
    position: absolute;
    top: 10px;
    right: 10px;
    cursor: pointer;
    font-size: 20px; /* Adjust size as needed */
  }
`;

  render() {
    return html`
    <div @click="${this.showPopover}" popovertarget="my-popover" popovertargetaction="show">ℹ️</div>
    <div popover="auto" class="popup" id="my-popover">
      <span @click="${this.closePopover}" class="close-button" popovertargetaction="hide">&times;</span>
      <slot></slot>
    </div>
    `;
  }

  showPopover() {
    this.popoverElement.showPopover();
  }

  closePopover() {
    if (this.popoverElement) {
      this.popoverElement.hidePopover();
    }
  }
}
