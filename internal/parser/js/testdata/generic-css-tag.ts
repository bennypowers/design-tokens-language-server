import { css, CSSResult } from 'lit';

const styles = css<CSSResult>`
  :host {
    display: block;
    color: var(--host-color);
  }
  .content {
    padding: var(--content-padding, 16px);
  }
`;
