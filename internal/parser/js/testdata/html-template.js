import { html } from 'lit';

export function render() {
  return html`
    <style>
      .card {
        color: var(--text-color);
        background: var(--card-bg, #fff);
      }
    </style>
    <div style="padding: var(--spacing)">
      <h1>Hello</h1>
    </div>
  `;
}
