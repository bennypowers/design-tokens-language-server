import { describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { cssColorToLspColor } from "#color";

import { createTestContext } from "#test-helpers";

import { colorPresentation } from "./colorPresentation.ts";

describe("textDocument/colorPresentation", () => {
  const ctx = createTestContext({
    testTokensSpecs: [
      {
        prefix: "token",
        spec: "file:///tokens.json",
        tokens: {
          color: {
            red: {
              _: {
                $value: "#ff0000",
                $type: "color",
              },
              hex: {
                $value: "#ff0000",
                $type: "color",
              },
              malformed: {
                $value: "ff 00 00",
                $type: "color",
              },
              wrongtype: {
                $value: "red",
                $type: "dimension",
              },
            },
          },
          space: {
            small: {
              $value: "4px",
              $type: "dimension",
            },
          },
        },
      },
    ],
  });
  const textDocument = ctx.documents.createCssDocument(/*css*/ `
    a {
      color: var(--token-color-red);
      color: var(--token-color-red-malformed);
      color: var(--token-color-red-wrongtype);
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
