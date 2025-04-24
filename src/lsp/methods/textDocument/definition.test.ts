import { describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { createTestContext } from "#test-helpers";

import { definition } from "./definition.ts";

describe("textDocument/definition", () => {
  const spec = "file:///tokens.json";
  const ctx = createTestContext({
    testTokensSpecs: [{
      prefix: "token",
      spec,
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
        },
        space: {
          small: {
            $value: "4px",
            $type: "size",
          },
        },
      },
    }],
  });

  describe("in a css document", () => {
    const textDocument = ctx.documents.createCssDocument(/*css*/ `
      a {
        color: var(--token-color-red);
        border-color: var(--token-color-red-hex);
        border-width: var(--token-space-small);
        handedness: var(--token-sinister);
      }
    `);

    const doc = ctx.documents.get(textDocument.uri);

    it("returns color presentation for a known token name", () => {
      const range = doc.rangeForSubstring("--token-color-red");
      const position = range.start;
      expect(definition({ textDocument, position }, ctx)).toEqual([
        {
          uri: spec,
          range: {
            start: { line: 2, character: 11 },
            end: { line: 11, character: 5 },
          },
        },
      ]);
    });

    it("returns matching range for nested token", () => {
      const range = doc.rangeForSubstring("--token-color-red-hex");
      const position = range.start;
      expect(definition({ textDocument, position }, ctx)).toEqual([
        {
          uri: spec,
          range: {
            start: { line: 7, character: 13 },
            end: { line: 10, character: 7 },
          },
        },
      ]);
    });

    it("returns an empty list for undeclared tokens", () => {
      const range = doc.rangeForSubstring("--token-sinister");
      const location = definition(
        { textDocument, position: range.start },
        ctx,
      );
      expect(location).toEqual([]);
    });
  });

  describe("in a json document", () => {
    const textDocument = ctx.documents.createJsonDocument(/*json*/ `
      {
        "color": {
          "red": {
            "_": {
              "$value": "#ff0000",
              "$type": "color"
            },
            "hex": {
              "$value": "{color.red._}",
              "$type": "color"
            }
          }
        }
      }
    `);

    it("throws", () => {
      expect(() =>
        definition({
          textDocument,
          position: { line: 2, character: 11 },
        }, ctx)
      ).toThrow(
        "textDocument/definition not implemented for JSON documents",
      );
    });
  });
});
