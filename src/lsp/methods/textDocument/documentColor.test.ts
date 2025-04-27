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
            $type: "color",
            red: {
              _: {
                $value: "#ff0000",
              },
              hex: {
                $value: "#ff0000",
              },
            },
            blue: {
              light: {
                $value: "#0000ff",
              },
              dark: {
                $value: "darkblue",
              },
              lightdark: {
                $value:
                  "light-dark(var(--token-blue-light, #0000ff), var(--token-blue-dark, darkblue))",
              },
              reference: {
                $value: "{color.blue.light}",
              },
              callreference: {
                $value: "light-dark({color.blue.light}, {color.blue.dark})",
              },
            },
          },
          space: {
            $type: "size",
            small: {
              $value: "4px",
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
      const doc = ctx.documents.get(textDocument.uri);

      it("should return a single DocumentColor", () => {
        const results = documentColor({ textDocument }, ctx);
        expect(results).not.toBeNull();
        expect(results).toHaveLength(1);
        const [result] = results;
        expect(result.color).toEqual(cssColorToLspColor("red"));
        expect(result.range).toEqual(
          doc.rangeForSubstring("--token-color-red"),
        );
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
        const doc = ctx.documents.get(textDocument.uri);
        expect(results).not.toBeNull();
        expect(results).toHaveLength(1);
        const [result] = results;
        expect(result.color).toEqual(cssColorToLspColor("red"));
        expect(result.range).toEqual(
          doc.rangeForSubstring("--token-color-red"),
        );
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
        const doc = ctx.documents.get(textDocument.uri);
        const range = doc.rangeForSubstring(
          "--token-color-blue-lightdark",
        );
        expect(results).toEqual([
          {
            color: cssColorToLspColor("#0000ff"),
            range,
          },
          {
            color: cssColorToLspColor("darkblue"),
            range,
          },
        ]);
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
