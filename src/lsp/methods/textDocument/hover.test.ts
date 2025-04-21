import { describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { MarkupContent } from "vscode-languageserver-protocol";

import { createTestContext } from "#test-helpers";

import { hover } from "./hover.ts";

describe("textDocument/hover", () => {
  const ctx = createTestContext();

  it("should return hover information for a token", () => {
    const textDocument = ctx.documents.create(/*css*/ `
      a {
        color:var(--token-color-red);
      }
    `);

    const position = textDocument.positionOf("--token-color-red");

    const result = hover({ textDocument, position }, ctx);

    expect(result).not.toBeNull();
    expect(result?.range).toEqual(textDocument.rangeOf("--token-color-red"));
    expect(result?.contents).toHaveProperty("kind", "markdown");
    expect(result?.contents).toHaveProperty("value");
    expect((result?.contents as MarkupContent).value).toEqual(`\
      # \`--token-color-red\`

      Type: \`color\`
      Red colour

      \`\`\`css
      red
      \`\`\``.replaceAll(/^ {6}/gm, ""));
  });

  it("should return null for a non-token", () => {
    const textDocument = ctx.documents.create(/*css*/ `
      a {
        color: :var(--non-token);
      }
    `);

    const position = textDocument.positionOf("--non-token");

    const result = hover({ textDocument, position }, ctx);

    expect(result).toBeNull();
  });

  it("should return formatted hover information for a token with a light-dark value", () => {
    const textDocument = ctx.documents.create(/*css*/ `
      a{
        color: var(--token-color-blue-lightdark);
      }
    `);

    const position = textDocument.positionOf("--token-color-blue-lightdark");

    const range = textDocument.rangeOf("--token-color-blue-lightdark");

    const result = hover({ textDocument, position }, ctx);

    expect(result).not.toBeNull();
    expect(result?.range).toEqual(range);
    expect(result?.contents).toHaveProperty("kind", "markdown");
    expect(result?.contents).toHaveProperty("value");
    expect((result?.contents as MarkupContent).value).toEqual(`\
      # \`--token-color-blue-lightdark\`

      Type: \`color\`
      Color scheme color

      \`\`\`css
      color: light-dark(
        lightblue,
        darkblue
      )
      \`\`\``.replaceAll(/^ {6}/gm, ""));
  });
});
