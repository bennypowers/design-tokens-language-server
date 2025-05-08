import { beforeEach, describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { cssColorToLspColor } from "#color";

import { createTestContext, DTLSTestContext } from "#test-helpers";

import { colorPresentation } from "./colorPresentation.ts";
import { ColorPresentation } from "vscode-languageserver-protocol";

describe("textDocument/colorPresentation", () => {
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
                malformed: {
                  $value: "ff 00 00",
                },
                wrongtype: {
                  $value: "red",
                  $type: "dimension",
                },
              },
            },
            space: {
              $type: "dimension",
              small: {
                $value: "4px",
              },
            },
          },
        },
      ],
    });
  });

  describe("in a document with tokens and non-tokens", () => {
    let result: ColorPresentation[];
    beforeEach(() => {
      const textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
          a {
            color: var(--token-color-red);
            color: var(--token-color-red-malformed);
            color: var(--token-color-red-wrongtype);
            border-color: var(--token-color-red-hex);
            border-width: var(--token-space-small);
          }
        `,
      );
      const doc = ctx.documents.get(textDocument.uri);
      const range = doc.getRangeForSubstring("--token-color-red");
      const color = cssColorToLspColor("red")!;
      result = colorPresentation({ textDocument, color, range }, ctx);
    });
    it("should return color presentations for matching colors", () => {
      expect(result).toEqual([
        { label: "--token-color-red" },
        { label: "--token-color-red-hex" },
      ]);
    });
  });
});
