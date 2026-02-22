import { css, html } from 'lit';

const styles = css`
  .card {
    color: var(--text-color);
    background: var(--card-bg, #fff);
  }
`;

export function Card({ children }) {
  return (
    <div className="card" style={{ color: 'var(--text-color)' }}>
      {children}
    </div>
  );
}
