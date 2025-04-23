import { describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { cssColorToLspColor } from "#color";

import { createTestContext } from "#test-helpers";

import { documentColor } from "./documentColor.ts";

describe("textDocument/documentColor", () => {
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
            },
            blue: {
              lightdark: {
                $value: "light-dark(#0000ff, darkblue)",
                $type: "color",
              },
            },
          },
          space: {
            small: {
              $value: "4px",
              $type: "size",
            },
          },
        },
      },
    ],
  });

  describe("in a css document", () => {
    describe("with a single token with type color", () => {
      const textDocument = ctx.documents.createCssDocument(/*css*/ `
      a {
        color: var(--token-color-red);
      }
    `);

      it("should return a single DocumentColor", () => {
        const results = documentColor({ textDocument }, ctx);
        expect(results).not.toBeNull();
        expect(results).toHaveLength(1);
        const [result] = results;
        expect(result.color).toEqual(cssColorToLspColor("red"));
        expect(result.range).toEqual(textDocument.rangeOf("--token-color-red"));
      });
    });

    describe("with a single token with type dimension", () => {
      const textDocument = ctx.documents.createCssDocument(/*css*/ `
      a{
        color: var(--token-space-small)
      }
    `);

      it("should return an empty array", () => {
        const results = documentColor({ textDocument }, ctx);
        expect(results).toHaveLength(0);
      });
    });

    describe("with two tokens: one color, one dimension", () => {
      const textDocument = ctx.documents.createCssDocument(/*css*/ `
      a {
        color: var(--token-color-red);
        width: var(--token-space-small);
      }
    `);

      it("should return a single DocumentColor", () => {
        const results = documentColor({ textDocument }, ctx);
        expect(results).not.toBeNull();
        expect(results).toHaveLength(1);
        const [result] = results;
        expect(result.color).toEqual(cssColorToLspColor("red"));
        expect(result.range).toEqual(textDocument.rangeOf("--token-color-red"));
      });
    });

    describe("with a single token with type color and light-dark values", () => {
      const textDocument = ctx.documents.createCssDocument(/*css*/ `
      a {
        color: var(--token-color-blue-lightdark);
      }
    `);

      it("should return two DocumentColors for the same token", () => {
        const results = documentColor({ textDocument }, ctx);
        const range = textDocument.rangeOf("--token-color-blue-lightdark");
        expect(results).not.toBeNull();
        expect(results).toHaveLength(2);
        const [result1, result2] = results;
        expect(result1.color).toEqual(cssColorToLspColor("#0000ff"));
        expect(result2.color).toEqual(cssColorToLspColor("darkblue"));
        expect(result1.range).toEqual(range);
        expect(result2.range).toEqual(range);
      });
    });
  });

  describe("in a json document", () => {
    const textDocument = ctx.documents.createJsonDocument(/*json*/ `
      {
        "color": "var(--token-color-red)"
      }
    `);

    it("should return an empty array", () => {
      const results = documentColor({ textDocument }, ctx);
      expect(results).toHaveLength(0);
    });
  });
});
