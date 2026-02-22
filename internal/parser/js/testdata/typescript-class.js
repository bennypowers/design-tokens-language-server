import { LitElement, html, css } from 'lit';

class MyElement extends LitElement {
  static styles = css`
    :host {
      display: block;
      color: var(--host-color);
    }
    .content {
      padding: var(--content-padding, 16px);
    }
  `;

  render() {
    return html`
      <style>
        .inner {
          margin: var(--inner-margin);
        }
      </style>
      <div class="content">
        <slot></slot>
      </div>
    `;
  }
}
