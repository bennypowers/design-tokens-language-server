import { beforeEach, describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { cssColorToLspColor } from "#color";

import { createTestContext, DTLSTestContext } from "#test-helpers";

import { documentColor } from "./documentColor.ts";
import { CssDocument } from "#css";
import { TextDocumentIdentifier } from "vscode-languageserver-protocol";
import { JsonDocument } from "#json";

describe("textDocument/documentColor", () => {
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
  });

  describe("in a css document", () => {
    describe("with a single token with type color", () => {
      let textDocument: TextDocumentIdentifier;
      let doc: CssDocument;
      beforeEach(() => {
        textDocument = ctx.documents.createDocument(
          "css",
          /*css*/ `
            a {
              color: var(--token-color-red);
            }
          `,
        );
        doc = ctx.documents.get(textDocument.uri) as CssDocument;
      });
      it("should return a single DocumentColor", () => {
        const results = documentColor({ textDocument }, ctx);
        expect(results).not.toBeNull();
        expect(results).toHaveLength(1);
        const [result] = results;
        expect(result.color).toEqual(cssColorToLspColor("red"));
        expect(result.range).toEqual(
          doc.getRangeForSubstring("--token-color-red"),
        );
      });
    });

    describe("with a single token with type dimension", () => {
      let textDocument: TextDocumentIdentifier;
      beforeEach(() => {
        textDocument = ctx.documents.createDocument(
          "css",
          /*css*/ `
            a{
              color: var(--token-space-small)
            }
          `,
        );
      });
      it("should return an empty array", () => {
        const results = documentColor({ textDocument }, ctx);
        expect(results).toHaveLength(0);
      });
    });

    describe("with two tokens: one color, one dimension", () => {
      let textDocument: TextDocumentIdentifier;
      let doc: CssDocument;
      beforeEach(() => {
        textDocument = ctx.documents.createDocument(
          "css",
          /*css*/ `
            a {
              color: var(--token-color-red);
              width: var(--token-space-small);
            }
          `,
        );
        doc = ctx.documents.get(textDocument.uri) as CssDocument;
      });
      it("should return a single DocumentColor", () => {
        const results = documentColor({ textDocument }, ctx);
        expect(results).not.toBeNull();
        expect(results).toHaveLength(1);
        const [result] = results;
        expect(result.color).toEqual(cssColorToLspColor("red"));
        expect(result.range).toEqual(
          doc.getRangeForSubstring("--token-color-red"),
        );
      });
    });

    describe("with a single token with type color and light-dark values", () => {
      let textDocument: TextDocumentIdentifier;
      let doc: CssDocument;
      beforeEach(() => {
        textDocument = ctx.documents.createDocument(
          "css",
          /*css*/ `
            a {
              color: var(--token-color-blue-lightdark);
            }
          `,
        );
        doc = ctx.documents.get(textDocument.uri) as CssDocument;
      });
      it("should return two DocumentColors for the same token", () => {
        const results = documentColor({ textDocument }, ctx);
        const range = doc.getRangeForSubstring("--token-color-blue-lightdark");
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

  describe("in a json document with 2 internal references to 2 colors", () => {
    let textDocument: TextDocumentIdentifier;
    let doc: JsonDocument;
    beforeEach(() => {
      textDocument = ctx.documents.createDocument(
        "json",
        /*json*/ `
          {
            "color": {
              "$type": "color",
              "blue": {
                "light": { "$value": "#0000ff" },
                "dark": { "$value": "darkblue" },
                "lightdark": { "$value": "light-dark({color.blue.light}, {color.blue.dark})" }
              }
            },
            "noncolor": {
              "$type": "dimension",
              "small": { "$value": "red" }
            }
          }
        `,
      );
      doc = ctx.documents.get(textDocument.uri) as JsonDocument;
    });

    it("should return 4 colors", () => {
      const results = documentColor({ textDocument }, ctx);
      expect(results).toEqual([
        {
          color: cssColorToLspColor("#0000ff"),
          range: doc.getRangeForSubstring("#0000ff"),
        },
        {
          color: cssColorToLspColor("darkblue"),
          range: doc.getRangeForSubstring("darkblue"),
        },
        {
          color: cssColorToLspColor("#0000ff"),
          range: doc.getRangeForSubstring("color.blue.light"),
        },
        {
          color: cssColorToLspColor("darkblue"),
          range: doc.getRangeForSubstring("color.blue.dark"),
        },
      ]);
    });
  });
});
