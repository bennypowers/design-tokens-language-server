import { beforeEach, describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import * as LSP from "vscode-languageserver-protocol";

import { createTestContext, DTLSTestContext } from "#test-helpers";

import { completion } from "./completion.ts";

describe("textDocument/completion", () => {
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
            },
            space: {
              $type: "size",
              small: {
                $value: "4px",
              },
            },
            font: {
              weight: {
                $type: "fontWeight",
                thin: {
                  $value: 100,
                },
              },
            },
          },
        },
      ],
    });
  });

  describe("in an empty document", () => {
    let completions: LSP.CompletionList | null;
    beforeEach(() => {
      completions = completion({
        textDocument: ctx.documents.createDocument("css", ""),
        position: { line: 0, character: 0 },
      }, ctx);
    });
    it("should return no completions", () => {
      expect(completions).toBeNull();
    });
  });

  describe("in a css document with an incomplete rule", () => {
    let completions: LSP.CompletionList | null;
    beforeEach(() => {
      const textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
          body {
            a`,
      );
      const doc = ctx.documents.get(textDocument.uri);
      completions = completion({
        textDocument,
        position: doc.getRangeForSubstring("a").end,
      }, ctx);
    });
    it("should return no completions", () => {
      expect(completions).toBeNull();
    });
  });

  describe("adding the token prefix in a malformed block", () => {
    let completions: LSP.CompletionList | null;
    beforeEach(() => {
      const textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
        body {
          token
        }
      `,
      );
      const doc = ctx.documents.get(textDocument.uri);
      completions = completion({
        textDocument,
        position: doc.getRangeForSubstring("token").end,
      }, ctx);
    });
    it("should return all token completions", () => {
      expect(completions?.items).toHaveLength(ctx.tokens.size);
    });
  });

  describe("adding the token prefix as a property name", () => {
    let completions: LSP.CompletionList | null;
    beforeEach(() => {
      const textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
          body {
            --token
          }
        `,
      );
      const doc = ctx.documents.get(textDocument.uri);
      completions = completion({
        textDocument,
        position: doc.getRangeForSubstring("--token").end,
      }, ctx);
    });
    it("should return all token completions", () => {
      expect(completions?.items).toHaveLength(ctx.tokens.size);
    });
    it("should return token completions as property names", () => {
      for (const item of completions?.items ?? []) {
        expect(item.textEdit?.newText).toMatch(/^--token/);
      }
    });
  });

  describe("adding the token prefix as a property value", () => {
    let completions: LSP.CompletionList | null;
    beforeEach(() => {
      const textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
          body {
            color: token
          }
        `,
      );
      const doc = ctx.documents.get(textDocument.uri);
      completions = completion({
        textDocument,
        position: doc.getRangeForSubstring("token").end,
      }, ctx);
    });
    it("should return all token completions", () => {
      expect(completions?.items).toHaveLength(ctx.tokens.size);
    });
    it("should return token completions as var() calls", () => {
      for (const item of completions?.items ?? []) {
        expect(item.textEdit?.newText).toMatch(/^var\(--token/);
      }
    });
  });

  describe("in a yaml file", () => {
    let completions: LSP.CompletionList | null;
    describe("opening a string property", () => {
      beforeEach(() => {
        const textDocument = ctx.documents.createDocument(
          "yaml",
          /*yaml*/ `
            color:
              $type: color
              bread:
                $value: '`,
        );
        const doc = ctx.documents.get(textDocument.uri);
        completions = completion({
          textDocument,
          position: doc.getRangeForSubstring("$value: '").end,
        }, ctx);
      });
      it("should return all token completions", () => {
        expect(completions?.items)
          .toHaveLength(
            ctx.tokens
              .values()
              .toArray()
              .length,
          );
      });
      it("should return token completions as references", () => {
        for (const item of completions?.items ?? []) {
          expect(item.label).toMatch(/^'\{.*}'$/);
        }
      });
      it("should return token names as contextual data for completions", () => {
        for (const item of completions?.items ?? []) {
          expect(item.data?.tokenName).toMatch(/^--token-/);
        }
      });
    });
    describe("prefixing `{c`", () => {
      beforeEach(() => {
        const textDocument = ctx.documents.createDocument(
          "yaml",
          /*yaml*/ `
            color:
              $type: color
              bread:
                $value: '{co`,
        );
        const doc = ctx.documents.get(textDocument.uri);
        const { end } = doc.getRangeForSubstring("{co");
        completions = completion({
          textDocument,
          position: end,
        }, ctx);
      });
      it("should return only color token completions", () => {
        expect(completions?.items)
          .toHaveLength(
            ctx.tokens
              .values()
              .filter((x) => x.$type === "color")
              .toArray()
              .length,
          );
      });
    });
  });

  describe("in a json file", () => {
    let completions: LSP.CompletionList | null;
    describe("opening a string property", () => {
      beforeEach(() => {
        const textDocument = ctx.documents.createDocument(
          "json",
          /*json*/ `
            {
              "color": {
                "$type": "color",
                "bread": {
                  "$value": ""
                }
              }
            }`,
        );
        const doc = ctx.documents.get(textDocument.uri);
        completions = completion({
          textDocument,
          position: doc.getRangeForSubstring('"$value": "').end,
        }, ctx);
      });
      it("should return all token completions", () => {
        expect(completions?.items)
          .toHaveLength(
            ctx.tokens
              .values()
              .toArray()
              .length,
          );
      });
      it("should return token completions as references", () => {
        for (const item of completions?.items ?? []) {
          expect(item.label).toMatch(/^"\{.*}"$/);
        }
      });
      it("should return token names as contextual data for completions", () => {
        for (const item of completions?.items ?? []) {
          expect(item.data?.tokenName).toMatch(/^--token-/);
        }
      });
    });
    describe("prefixing `{c`", () => {
      beforeEach(() => {
        const textDocument = ctx.documents.createDocument(
          "json",
          /*json*/ `
            {
              "color": {
                "$type": "color",
                "bread": {
                  "$value": "{c"
                }
              }
            }`,
        );
        const doc = ctx.documents.get(textDocument.uri);
        completions = completion({
          textDocument,
          position: doc.getRangeForSubstring("{c").end,
        }, ctx);
      });
      it("should return only color token completions", () => {
        expect(completions?.items)
          .toHaveLength(
            ctx.tokens
              .values()
              .filter((x) => x.$type === "color")
              .toArray()
              .length,
          );
      });
    });
  });
});
