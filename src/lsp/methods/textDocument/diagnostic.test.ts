import { describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { createTestContext } from "#test-helpers";

import { DTLSErrorCodes } from "#lsp";

import { diagnostic } from "./diagnostic.ts";

describe("textDocument/diagnostic", () => {
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
                $value: "red",
                $description: "Red colour",
              },
              hex: {
                $value: "#ff0000",
              },
            },
            blue: {
              lightdark: {
                $value: "light-dark(lightblue, darkblue)",
                $description: "Color scheme color",
              },
            },
          },
          space: {
            $type: "size",
            small: {
              $value: "4px",
            },
          },
          font: {
            family: {
              $value: "'Super Duper', Helvetica, Arial, sans-serif",
              $type: "fontFamily",
            },
            mishpocha: {
              $value: "Super, 'Pooper Duper', Helvetica, Arial, sans-serif",
              $type: "fontFamily",
            },
            weight: {
              $value: 400,
              $type: "fontWeight",
            },
            heft: {
              $value: "400",
            },
          },
        },
      },
    ],
  });

  describe("in a document with a single token and no fallback", () => {
    const textDocument = ctx.documents.createCssDocument(/*css*/ `
      body { color: var(--token-color); }
    `);
    it("should return no diagnostics", () => {
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toHaveLength(0);
    });
  });

  describe("in a document with a single token and an incorrect fallback", () => {
    const textDocument = ctx.documents.createCssDocument(/*css*/ `
      body {
        color: var(--token-color-red, blue);
      }
    `);
    it("should return a single diagnostic", () => {
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toHaveLength(1);
      const [diag] = diagnostics.items;
      expect(diag.message).toBe(
        "Token fallback does not match expected value: red",
      );
    });
  });

  describe("in a document with a single token and an incorrect fallback list", () => {
    const textDocument = ctx.documents.createCssDocument(/*css*/ `
      body {
        color: var(--token-color-red, blue, green, mango, goBuckWild);
      }
    `);
    it("should return a single diagnostic", () => {
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toHaveLength(1);
      const [diag] = diagnostics.items;
      expect(diag.message).toBe(
        "Token fallback does not match expected value: red",
      );
    });
  });

  describe("in a document with a single list-value token and an incorrect fallback list", () => {
    const textDocument = ctx.documents.createCssDocument(/*css*/ `
      body {
        color: var(--token-font-family, a, b, c, d);
      }
    `);
    it("should return a single diagnostic", () => {
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toHaveLength(1);
      const [diag] = diagnostics.items;
      expect(diag.message).toBe(
        "Token fallback does not match expected value: 'Super Duper', Helvetica, Arial, sans-serif",
      );
    });
  });

  describe("in a document with a saucy font-family", () => {
    const textDocument = ctx.documents.createCssDocument(/*css*/ `
      body {
        font-family: var(--token-font-mishpocha, Super, 'Pooper Duper', Helvetica, Arial, sans-serif);
      }
    `);
    it("should return no diagnostics", () => {
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toHaveLength(0);
    });
  });

  describe("in a document with a single number-value token and a correct fallback", () => {
    const textDocument = ctx.documents.createCssDocument(/*css*/ `
      body {
        color: var(--token-font-weight, 400);
      }
    `);
    it("should return an empty list", () => {
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toEqual([]);
    });
  });

  describe("in a document with a single stringy-number token and correct fallback", () => {
    const textDocument = ctx.documents.createCssDocument(/*css*/ `
      body {
        color: var(--token-font-heft, '400');
      }
    `);
    it("should return a single diagnostic list", () => {
      const doc = ctx.documents.get(textDocument.uri)!;
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toEqual([{
        code: DTLSErrorCodes.incorrectFallback,
        severity: 1,
        data: {
          tokenName: "--token-font-heft",
        },
        message: "Token fallback does not match expected value: 400",
        range: doc.getRangeForSubstring("'400'"),
      }]);
    });
  });

  describe("in a document with a single stringy-number token and number fallback", () => {
    const textDocument = ctx.documents.createCssDocument(/*css*/ `
      body {
        color: var(--token-font-heft, 400);
      }
    `);
    it("should return an empty list", () => {
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toEqual([]);
    });
  });

  describe("in a document with a single box-shadow token and accurate fallback", () => {
    const textDocument = ctx.documents.createCssDocument(/*css*/ `
      body {
        box-shadow: var(--token-box-shadow, 1px 2px 3px 4px rgba(2, 4, 6 / .8));
      }
    `);
    it("should return an empty list", () => {
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toEqual([]);
    });
  });
});
