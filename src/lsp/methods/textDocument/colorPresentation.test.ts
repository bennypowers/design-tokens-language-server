import { describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { cssColorToLspColor } from "#color";

import { TestDocuments, TestTokens } from "#test-helpers";

import { colorPresentation } from "./colorPresentation.ts";

const documents = new TestDocuments();
const tokens = new TestTokens();

describe("colorPresentation", () => {
  const uri = "file:///test.css";

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
