// NOTE: Line/column-based tests depend on exact layout. Do not reformat.
import { css } from 'lit';

export const styles = css`
  :root {
    --color-primary: #0000ff;
  }
  .button {
    color: var(--color-primary);
    background: var(--bg-color, #fff);
  }
`;
