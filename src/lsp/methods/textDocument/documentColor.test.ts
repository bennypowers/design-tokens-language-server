import * as LSP from "vscode-languageserver-protocol";

import { beforeEach, describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { cssColorToLspColor } from "#color";

import { createTestContext, DTLSTestContext } from "#test-helpers";

import { documentColor } from "./documentColor.ts";
import { CssDocument } from "#css";
import { JsonDocument } from "#json";
import { YamlDocument } from "#yaml";

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
      let doc: CssDocument;
      let results: LSP.ColorInformation[];
      beforeEach(() => {
        const textDocument = ctx.documents.createDocument(
          "css",
          /*css*/ `
            a {
              color: var(--token-color-red);
            }
          `,
        );
        doc = ctx.documents.get(textDocument.uri);
        results = documentColor({ textDocument }, ctx);
      });
      it("should return a single DocumentColor", () => {
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
      let results: LSP.ColorInformation[];
      beforeEach(() => {
        results = documentColor({
          textDocument: ctx.documents.createDocument(
            "css",
            /*css*/ `
            a{
              color: var(--token-space-small)
            }
          `,
          ),
        }, ctx);
      });
      it("should return an empty array", () => {
        expect(results).toHaveLength(0);
      });
    });

    describe("with two tokens: one color, one dimension", () => {
      let doc: CssDocument;
      let results: LSP.ColorInformation[];
      beforeEach(() => {
        const textDocument = ctx.documents.createDocument(
          "css",
          /*css*/ `
            a {
              color: var(--token-color-red);
              width: var(--token-space-small);
            }
          `,
        );
        doc = ctx.documents.get(textDocument.uri);
        results = documentColor({ textDocument }, ctx);
      });
      it("should return a single DocumentColor", () => {
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
      let doc: CssDocument;
      let results: LSP.ColorInformation[];
      beforeEach(() => {
        const textDocument = ctx.documents.createDocument(
          "css",
          /*css*/ `
            a {
              color: var(--token-color-blue-lightdark);
            }
          `,
        );
        doc = ctx.documents.get(textDocument.uri);
        results = documentColor({ textDocument }, ctx);
      });
      it("should return two DocumentColors for the same token", () => {
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
    let results: LSP.ColorInformation[];
    let doc: JsonDocument;
    beforeEach(() => {
      const textDocument = ctx.documents.createDocument(
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
      doc = ctx.documents.get(textDocument.uri);
      results = documentColor({ textDocument }, ctx);
    });

    it("should return 4 colors", () => {
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

  describe("in a yaml document that contains non-color values that look like colors", () => {
    let doc: YamlDocument;
    let results: LSP.ColorInformation[];
    beforeEach(() => {
      const textDocument = ctx.documents.createDocument(
        "yaml",
        /*yaml*/ `
            noncolor:
              $type: dimension
              small:
                $value: red
              green:
                $value: 1px
            color:
              $type: color
              purple:
                $value: rebeccapurple
                $description: sentimental color
        `,
      );
      doc = ctx.documents.get(textDocument.uri);
      results = documentColor({ textDocument }, ctx);
    });

    it("should return only 1 color", () => {
      expect(results).toEqual([{
        range: doc.getRangeForSubstring("rebeccapurple"),
        color: cssColorToLspColor("rebeccapurple"),
      }]);
    });
  });
});
