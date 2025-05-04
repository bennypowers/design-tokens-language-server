import { TextDocumentIdentifier } from "vscode-languageserver-protocol";

import { afterEach, beforeEach, describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { createTestContext, DTLSTestContext } from "#test-helpers";

import { definition } from "./definition.ts";
import { JsonDocument } from "#json";

describe("textDocument/definition", () => {
  describe("in a css document", () => {
    const spec = "/tokens.json";
    let ctx: DTLSTestContext;
    let textDocument: TextDocumentIdentifier;
    beforeEach(async () => {
      ctx = await createTestContext({
        testTokensSpecs: [{
          prefix: "token",
          spec,
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
          },
        }],
      });
      textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
        a {
          color: var(--token-color-red);
          border-color: var(--token-color-red-hex);
          border-width: var(--token-space-small);
          handedness: var(--token-sinister);
        }
      `,
      );
    });

    afterEach(() => ctx.clear());

    it("returns definition for a known token name", () => {
      const definitionUri = "file:///tokens.json";
      const doc = ctx.documents.get(textDocument.uri);
      const definitionDoc = ctx.documents.get(definitionUri);
      const range = doc.getRangeForSubstring("--token-color-red");
      const position = range.start;
      expect(definition({ textDocument, position }, ctx)).toEqual([
        {
          uri: definitionUri,
          range: definitionDoc.getRangeForPath(["color", "red", "_"]),
        },
      ]);
    });

    it("returns matching range for nested token", () => {
      const doc = ctx.documents.get(textDocument.uri);
      const definitionUri = "file:///tokens.json";
      const jsonDoc = ctx.documents.get(definitionUri) as JsonDocument;
      const position = doc.getRangeForSubstring("--token-color-red-hex").start;
      const result = definition({ textDocument, position }, ctx);
      expect(result).toEqual([
        {
          uri: definitionUri,
          range: jsonDoc.getRangeForPath(["color", "red", "hex"]),
        },
      ]);
    });

    it("returns an empty list for undeclared tokens", () => {
      const doc = ctx.documents.get(textDocument.uri);
      const range = doc.getRangeForSubstring("--token-sinister");
      const location = definition(
        { textDocument, position: range.start },
        ctx,
      );
      expect(location).toEqual([]);
    });
  });

  describe("in a json document", () => {
    let ctx: DTLSTestContext;
    let textDocument: { uri: string };

    beforeEach(async () => {
      ctx = await createTestContext({
        testTokensSpecs: [
          {
            spec: "/tokens-single-file.json",
            tokens: {
              color: {
                $type: "color",
                red: {
                  _: { $value: "#ff0000" },
                  hex: { $value: "{color.red._}" },
                },
              },
            },
          },
        ],
      });
      textDocument = { uri: "file:///tokens-single-file.json" };
    });

    afterEach(() => ctx.clear());

    it("does not throw", () => {
      expect(() =>
        definition({
          textDocument,
          position: { line: 2, character: 11 },
        }, ctx)
      ).not.toThrow("hi");
    });

    it("returns a reference within the document", () => {
      const doc = ctx.documents.get(textDocument.uri) as JsonDocument;
      const position = doc.getRangeForSubstring("color.red._").start;
      const result = definition({ textDocument, position }, ctx);
      expect(result).toEqual([{
        uri: textDocument.uri,
        range: doc.getRangeForPath(["color", "red", "_"]),
      }]);
    });
  });

  describe("with multiple json documents", () => {
    let ctx: DTLSTestContext;
    let referee: JsonDocument;
    let referer: JsonDocument;
    beforeEach(async () => {
      ctx = await createTestContext({
        testTokensSpecs: [
          {
            spec: "/referee.json",
            tokens: {
              color: {
                $type: "color",
                red: {
                  $value: "#ff0000",
                  $description: "Red color",
                },
              },
            },
          },
          {
            spec: "/referer.json",
            tokens: {
              color: {
                $type: "color",
                roit: {
                  $value: "{color.red}",
                  $description: "Red color",
                },
              },
            },
          },
        ],
      });
      referee = ctx.documents.get("file:///referee.json") as JsonDocument;
      referer = ctx.documents.get("file:///referer.json") as JsonDocument;
    });

    afterEach(() => ctx.clear());

    it("returns a reference outside the document", () => {
      const position = referer.getRangeForSubstring("color.red").start;
      const result = definition({ textDocument: referer, position }, ctx);
      expect(result).toEqual([{
        uri: referee.uri,
        range: referee.getRangeForPath(["color", "red"]),
      }]);
    });
  });
});
