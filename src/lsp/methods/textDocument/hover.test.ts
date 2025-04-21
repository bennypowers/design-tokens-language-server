import { describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { MarkupContent } from "vscode-languageserver-protocol";
import { TestDocuments, TestTokens } from "#test-helpers";

import { hover } from "./hover.ts";

const css = String.raw;

describe("hover", () => {
  const tokens = new TestTokens();
  const documents = new TestDocuments(tokens);

  it("should return hover information for a token", () => {
    const uri = documents.create(css`a{b:var(--token-color-red)}\n`);

    const result = hover({
      textDocument: { uri },
      position: { line: 0, character: 10 },
    }, { documents, tokens });

    expect(result).not.toBeNull();
    expect(result?.contents).toHaveProperty("kind", "markdown");
    expect(result?.contents).toHaveProperty("value");
    expect((result?.contents as MarkupContent).value).toEqual(`\
      # \`--token-color-red\`

      Type: \`color\`
      Red colour

      \`\`\`css
      red
      \`\`\``.replaceAll(/^ {6}/gm, ""));
    expect(result?.range).toEqual({
      end: {
        character: 25,
        line: 0,
      },
      start: {
        character: 8,
        line: 0,
      },
    });
  });

  it("should return null for a non-token", () => {
    const uri = documents.create(css`a{b:var(--non-token)}\n`);

    const result = hover({
      textDocument: { uri },
      position: { line: 0, character: 10 },
    }, { documents, tokens });

    expect(result).toBeNull();
  });

  it("should return formatted hover information for a token with a light-dark value", () => {
    const uri = documents.create(css`a{b:var(--token-color-blue-lightdark)}`);

    const result = hover({
      textDocument: { uri },
      position: { line: 0, character: 10 },
    }, { documents, tokens });

    expect(result).not.toBeNull();
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
    expect(result?.range).toEqual({
      end: {
        character: 36,
        line: 0,
      },
      start: {
        character: 8,
        line: 0,
      },
    });
  });
});
