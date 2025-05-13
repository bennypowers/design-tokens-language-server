import * as LSP from "vscode-languageserver-protocol";

import { beforeEach, describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { createTestContext, DTLSTestContext } from "#test-helpers";

import { documentColor } from "./documentColor.ts";
import { CssDocument } from "#css";
import { JsonDocument } from "#json";
import { YamlDocument } from "#yaml";

describe("textDocument/documentSymbols", () => {
  let ctx: DTLSTestContext;

  beforeEach(async () => {
    ctx = await createTestContext({
      testTokensSpecs: [
        {
          prefix: "token",
          spec: "file:///tokens.json",
          tokens: {
            deprecated: {
              $type: "dimension",
              $deprecated: "because reasons",
              $value: "0px",
            },
          },
        },
      ],
    });
  });

  describe("in a css document", () => {
    describe("with a single deprecated token", () => {
      let doc: CssDocument;
      let results: LSP.ColorInformation[];
      beforeEach(() => {
        const textDocument = ctx.documents.createDocument(
          "css",
          /*css*/ `
            a {
              height: var(--token-deprecated);
            }
          `,
        );
        doc = ctx.documents.get(textDocument.uri);
        results = documentColor({ textDocument }, ctx);
      });

      it("returns a single deprecation symbol", () => {
        expect(results).toHaveLength(1);
        expect(results.at(0)).toEqual({
          range: doc.getRangeForSubstring("--token-deprecated"),
          selectionRange: doc.getRangeForSubstring("--token-deprecated"),
          name: "--token-deprecated",
          tags: [LSP.SymbolTag.Deprecated],
        });
      });
    });
  });
});
