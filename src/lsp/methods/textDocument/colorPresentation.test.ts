import { describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { cssColorToLspColor } from "#color";

import { TestDocuments, TestTokens } from "#test-helpers";

import { colorPresentation } from "./colorPresentation.ts";

const tokens = new TestTokens();
const documents = new TestDocuments(tokens);

const css = String.raw;

describe("colorPresentation", () => {
  const uri = documents.create(
    css`
      a {
        color: var(--token-color-red);
        border-color: var(--token-color-red-hex);
        border-width: var(--token-space-small);
      }
    `,
  );

  it("should return color presentations for matching colors", () => {
    const result = colorPresentation({
      textDocument: { uri },
      color: cssColorToLspColor("red"),
      range: {
        start: { line: 0, character: 0 },
        end: { line: 0, character: 0 },
      },
    }, { documents, tokens });
    expect(result).toEqual([
      { label: "token-color-red" },
      { label: "token-color-red-hex" },
    ]);
  });
});
