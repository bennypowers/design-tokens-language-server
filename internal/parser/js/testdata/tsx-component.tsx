import { css, LitElement, html } from 'lit';

const styles = css`
  :host {
    display: block;
    color: var(--host-color);
  }
  .content {
    padding: var(--content-padding, 16px);
  }
`;

interface CardProps {
  title: string;
  children: React.ReactNode;
}

export function Card({ title, children }: CardProps) {
  return (
    <div className="content">
      <h1>{title}</h1>
      {children}
    </div>
  );
}
