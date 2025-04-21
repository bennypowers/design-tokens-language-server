import { describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { createTestContext } from "#test-helpers";

import { definition } from "./definition.ts";

describe("textDocument/definition", () => {
  const ctx = createTestContext();
  const textDocument = ctx.documents.create(/*css*/ `
    a {
      color: var(--token-color-red);
      border-color: var(--token-color-red-hex);
      border-width: var(--token-space-small);
    }
  `);

  it("should return color presentations for matching colors", async () => {
    const range = textDocument.rangeOf("--token-color-red");
    const position = await definition(
      { textDocument, position: range.start },
      ctx,
    );
    const uri = new URL("../../../../test/tokens.json", import.meta.url).href;
    expect(position).toEqual([
      {
        uri,
        range: {
          start: { line: 31, character: 11 },
          end: { line: 35, character: 5 },
        },
      },
    ]);
  });
});
