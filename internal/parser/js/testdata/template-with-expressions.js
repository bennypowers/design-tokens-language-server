import { css } from 'lit';

const primary = 'blue';

export const styles = css`
  .before {
    color: var(--color-before);
  }
  ${someOtherStyles}
  .after {
    background: var(--color-after);
  }
`;
