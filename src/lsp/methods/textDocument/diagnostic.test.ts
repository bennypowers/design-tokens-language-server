import { afterEach, beforeEach, describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { createTestContext, DTLSTestContext } from "#test-helpers";

import { DTLSErrorCodes } from "#lsp";

import { diagnostic } from "./diagnostic.ts";
import { JsonDocument } from "#json";

describe("textDocument/diagnostic", () => {
  let ctx: DTLSTestContext;

  beforeEach(async () => {
    ctx = await createTestContext({
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
        {
          prefix: "token",
          spec: "file:///referer.json",
          tokens: {
            color: {
              $type: "color",
              badref: {
                $value: "{color.reed.dark}",
                $description: "Bad reference",
              },
            },
          },
        },
        {
          prefix: "token",
          spec: "file:///referer-good.json",
          tokens: {
            color: {
              $type: "color",
              badref: {
                $value: "{color.red._}",
                $description: "Good reference",
              },
            },
          },
        },
      ],
    });
  });

  afterEach(() => {
    ctx.clear();
  });

  describe("in a CSS document with a single token and no fallback", () => {
    it("should return no diagnostics", () => {
      const textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
        body { color: var(--token-color); }
      `,
      );
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toHaveLength(0);
    });
  });

  describe("in a CSS document with a single token and an incorrect fallback", () => {
    it("should return a single diagnostic", () => {
      const textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
        body {
          color: var(--token-color-red, blue);
        }
      `,
      );
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toHaveLength(1);
      const [diag] = diagnostics.items;
      expect(diag.message).toBe(
        "Token fallback does not match expected value: red",
      );
    });
  });

  describe("in a CSS document with a single token and an incorrect fallback list", () => {
    it("should return a single diagnostic", () => {
      const textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
        body {
          color: var(--token-color-red, blue, green, mango, goBuckWild);
        }
      `,
      );
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toHaveLength(1);
      const [diag] = diagnostics.items;
      expect(diag.message).toBe(
        "Token fallback does not match expected value: red",
      );
    });
  });

  describe("in a CSS document with a single list-value token and an incorrect fallback list", () => {
    it("should return a single diagnostic", () => {
      const textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
        body {
          color: var(--token-font-family, a, b, c, d);
        }
      `,
      );
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toHaveLength(1);
      const [diag] = diagnostics.items;
      expect(diag.message).toBe(
        "Token fallback does not match expected value: 'Super Duper', Helvetica, Arial, sans-serif",
      );
    });
  });

  describe("in a CSS document with a saucy font-family", () => {
    it("should return no diagnostics", () => {
      const textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
        body {
          font-family: var(--token-font-mishpocha, Super, 'Pooper Duper', Helvetica, Arial, sans-serif);
        }
      `,
      );
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toHaveLength(0);
    });
  });

  describe("in a CSS document with a single number-value token and a correct fallback", () => {
    it("should return an empty list", () => {
      const textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
        body {
          color: var(--token-font-weight, 400);
        }
      `,
      );
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toEqual([]);
    });
  });

  describe("in a CSS document with a single stringy-number token and correct fallback", () => {
    it("should return a single diagnostic list", () => {
      const textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
        body {
          color: var(--token-font-heft, '400');
        }
      `,
      );
      const doc = ctx.documents.get(textDocument.uri)!;
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toEqual([{
        code: DTLSErrorCodes.incorrectFallback,
        severity: 1,
        data: {
          tokenName: "--token-font-heft",
          actual: "'400'",
          expected: '400',
        },
        message: "Token fallback does not match expected value: 400",
        range: doc.getRangeForSubstring("'400'"),
      }]);
    });
  });

  describe("in a CSS document with a single stringy-number token and number fallback", () => {
    it("should return an empty list", () => {
      const textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
        body {
          color: var(--token-font-heft, 400);
        }
      `,
      );
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toEqual([]);
    });
  });

  describe("in a CSS document with a single box-shadow token and accurate fallback", () => {
    it("should return an empty list", () => {
      const textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
        body {
          box-shadow: var(--token-box-shadow, 1px 2px 3px 4px rgba(2, 4, 6 / .8));
        }
      `,
      );
      const diagnostics = diagnostic({ textDocument }, ctx);
      expect(diagnostics.items).toEqual([]);
    });
  });

  describe("in a JSON document which references an existing token", () => {
    it("should return a single diagnostic", () => {
      const doc = ctx.documents.get(
        "file:///referer-good.json",
      ) as JsonDocument;
      const diagnostics = diagnostic({ textDocument: doc }, ctx);
      expect(diagnostics.items).toHaveLength(0);
    });
  });

  describe("in a JSON document which references a non-existent token", () => {
    it("should return a single diagnostic", () => {
      const doc = ctx.documents.get("file:///referer.json") as JsonDocument;
      const diagnostics = diagnostic({ textDocument: doc }, ctx);
      expect(diagnostics.items).toHaveLength(1);
      const [diag] = diagnostics.items;
      expect(diag.code).toBe(DTLSErrorCodes.unknownReference);
      expect(diag.message).toBe(
        "Token reference does not exist: {color.reed.dark}",
      );
    });
  });
});
