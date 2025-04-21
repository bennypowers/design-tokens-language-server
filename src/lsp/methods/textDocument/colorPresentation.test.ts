import { describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { cssColorToLspColor } from "#color";

import { createTestContext } from "#test-helpers";

import { colorPresentation } from "./colorPresentation.ts";

describe("colorPresentation", () => {
  const ctx = createTestContext();
  const textDocument = ctx.documents.create(/*css*/ `
    a {
      color: var(--token-color-red);
      border-color: var(--token-color-red-hex);
      border-width: var(--token-space-small);
    }
  `);

  it("should return color presentations for matching colors", () => {
    const range = textDocument.rangeOf("--token-color-red");
    const color = cssColorToLspColor("red");
    const result = colorPresentation({ textDocument, color, range }, ctx);
    expect(result).toEqual([
      { label: "token-color-red" },
      { label: "token-color-red-hex" },
    ]);
  });
});
