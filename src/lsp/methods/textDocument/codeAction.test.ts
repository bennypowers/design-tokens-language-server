import { beforeEach, describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { createTestContext, DTLSTestContext } from "#test-helpers";

import { codeAction, DTLSCodeAction } from "./codeAction.ts";

import { resolve } from "../textDocument/codeAction.ts";

import {
  CodeActionKind,
  Diagnostic,
  TextDocumentIdentifier,
} from "vscode-languageserver-protocol";
import { DTLSTextDocument } from "#document";
import { CssDocument } from "#css";
import { diagnostic } from "#methods/textDocument/diagnostic.ts";
import { TextDocumentIdentifierFor } from "#documents";

describe("textDocument/codeAction", () => {
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
                },
              },
              blue: {
                lightdark: {
                  $value: "light-dark(lightblue, darkblue)",
                  $description: "Color scheme color",
                },
              },
            },
          },
        },
      ],
    });
  });

  describe("in a css document with one token var call and no fallback", () => {
    let result: ReturnType<typeof codeAction>;
    let textDocument: TextDocumentIdentifierFor<"css">;
    let doc: CssDocument;
    let diagnostics: Diagnostic[];

    beforeEach(() => {
      textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
        a {
          color: var(--token-color-red);
        }
      `,
      );
      doc = ctx.documents.get(textDocument.uri);
      diagnostics = doc.getDiagnostics(ctx);
    });

    describe("called on the first character of the file", () => {
      beforeEach(() => {
        result = codeAction({
          textDocument,
          range: {
            start: { line: 0, character: 0 },
            end: { line: 0, character: 0 },
          },
          context: {
            diagnostics,
          },
        }, ctx);
      });
      it("should return null", () => {
        expect(result).toBeNull();
      });
    });

    describe("calling codeAction on the first character of the token name", () => {
      beforeEach(() => {
        const position = doc.getRangeForSubstring("--token-color-red").start;
        result = codeAction({
          textDocument,
          range: { start: position, end: position },
          context: { diagnostics: doc.getDiagnostics(ctx) },
        }, ctx);
      });
      it("should return a single-token refactor action", () => {
        const newText = "var(--token-color-red, red)";
        const range = doc.getRangeForSubstring("var(--token-color-red)");
        expect(result).toEqual([
          {
            title: DTLSCodeAction.toggleFallback,
            kind: CodeActionKind.RefactorRewrite,
            edit: {
              changes: {
                [textDocument.uri]: [
                  {
                    range,
                    newText,
                  },
                ],
              },
            },
          },
        ]);
      });
    });
  });

  describe("in a css document with one token var call that has an incorrect fallback", () => {
    let textDocument: TextDocumentIdentifierFor<"css">;
    let doc: CssDocument;

    beforeEach(() => {
      textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
          a {
            color: var(--token-color-red, blue);
          }
        `,
      );
      doc = ctx.documents.get(textDocument.uri);
    });

    describe("called on the first character of the fallback", () => {
      let result: ReturnType<typeof codeAction>;

      beforeEach(() => {
        const position = doc.getRangeForSubstring("blue").start;
        const diagnostics = doc.getDiagnostics(ctx);
        result = codeAction({
          textDocument,
          range: { start: position, end: position },
          context: { diagnostics },
        }, ctx);
      });

      it("should return a fix for the incorrect fallback", () => {
        const diagnostics = doc.getDiagnostics(ctx);
        expect(result?.at(0)).toEqual({
          title: DTLSCodeAction.fixFallback,
          kind: CodeActionKind.QuickFix,
          diagnostics,
          data: { textDocument },
        });
      });
      it("should not return a fixall", () => {
        expect(result?.find((x) => x.kind === CodeActionKind.SourceFixAll))
          .toBeUndefined();
      });
    });
  });

  describe("in a css document with one token var call that has a correct fallback", () => {
    let textDocument: TextDocumentIdentifierFor<"css">;
    let doc: CssDocument;

    beforeEach(() => {
      textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
          a {
            color: var(--token-color-red, red);
          }
        `,
      );
      doc = ctx.documents.get(textDocument.uri);
    });

    describe("called on the first character of the fallback", () => {
      let result: ReturnType<typeof codeAction>;

      beforeEach(() => {
        const position = doc.getRangeForSubstring("red").start;
        const diagnostics = doc.getDiagnostics(ctx);
        result = codeAction({
          textDocument,
          range: { start: position, end: position },
          context: { diagnostics },
        }, ctx);
      });

      it("should return a single refactor action for a token call with fallback", () => {
        const range = doc.getRangeForSubstring("var(--token-color-red, red)");
        const newText = "var(--token-color-red)";
        expect(result).toEqual([
          {
            title: DTLSCodeAction.toggleFallback,
            kind: CodeActionKind.RefactorRewrite,
            edit: {
              changes: {
                [textDocument.uri]: [
                  {
                    range,
                    newText,
                  },
                ],
              },
            },
          },
        ]);
      });
    });
  });

  describe("in a css document with two token var calls with incorrect fallbacks", () => {
    let textDocument: TextDocumentIdentifierFor<"css">;
    let doc: CssDocument;

    beforeEach(() => {
      textDocument = ctx.documents.createDocument(
        "css",
        /*css*/ `
          a {
            color: var(--token-color-red, blue);
            border-color: var(--token-color-blue-lightdark, green);
          }
        `,
      );
      doc = ctx.documents.get(textDocument.uri);
    });

    let result: ReturnType<typeof codeAction>;

    beforeEach(() => {
      const diagnostics = doc.getDiagnostics(ctx);
      result = codeAction({
        textDocument,
        range: { start: doc.positionAt(0), end: doc.positionAt(0) },
        context: { diagnostics },
      }, ctx);
    });

    it("should return two fixes for the incorrect fallbacks and a fixall fix", () => {
      const diagnostics = doc.getDiagnostics(ctx);
      expect(result).toEqual([
        {
          title: DTLSCodeAction.fixFallback,
          kind: CodeActionKind.QuickFix,
          data: { textDocument },
          diagnostics: [diagnostics[0]],
        },
        {
          title: DTLSCodeAction.fixFallback,
          kind: CodeActionKind.QuickFix,
          data: { textDocument },
          diagnostics: [diagnostics[1]],
        },
        {
          title: DTLSCodeAction.fixAllFallbacks,
          kind: CodeActionKind.SourceFixAll,
          data: { textDocument },
        },
      ]);
    });

    describe("then performing one of the fixes", () => {
      beforeEach(() => {
        const action = result?.find(({ kind }) =>
          kind === CodeActionKind.QuickFix
        );
        if (action) {
          const edits = resolve(action!, ctx)?.edit?.changes?.[doc.uri];
          if (edits) {
            const text = DTLSTextDocument.applyEdits(doc, edits);
            doc.update([{ text }], doc.version + 1);
          }
        }
      });

      it("fixes that part of the file", () => {
        const text = ctx.documents.get(textDocument.uri).getText();
        expect(text).toEqual(/*css*/ `
          a {
            color: var(--token-color-red, red);
            border-color: var(--token-color-blue-lightdark, green);
          }
        `);
      });
    });

    describe("then performing the fixall", () => {
      beforeEach(() => {
        const edits = resolve(
          result!
            .find((x) => x.kind === CodeActionKind.SourceFixAll)!,
          ctx,
        )?.edit?.changes?.[doc.uri];
        if (edits) {
          const text = DTLSTextDocument.applyEdits(doc, edits);
          doc.update([{ text }], doc.version + 1);
        }
      });

      it("fixes the file", () => {
        const text = ctx.documents.get(textDocument.uri).getText();
        expect(text).toEqual(/*css*/ `
          a {
            color: var(--token-color-red, red);
            border-color: var(--token-color-blue-lightdark, light-dark(lightblue, darkblue));
          }
        `);
      });

      describe("and then calling codeAction on the range inside the {}", () => {
        let result: ReturnType<typeof codeAction>;

        beforeEach(() => {
          const diagnostics = doc.getDiagnostics(ctx);
          result = codeAction({
            textDocument: doc.identifier,
            range: {
              start: doc.getRangeForSubstring("color:").start,
              end: doc.getRangeForSubstring("darkblue));").end,
            },
            context: { diagnostics },
          }, ctx);
        });

        it("should return a single fallback-toggle range refactor", () => {
          expect(result).toEqual([
            {
              kind: CodeActionKind.RefactorRewrite,
              title: DTLSCodeAction.toggleRangeFallbacks,
              edit: {
                changes: {
                  [textDocument.uri]: [
                    {
                      range: doc.getRangeForSubstring(
                        "var(--token-color-red, red)",
                      ),
                      newText: "var(--token-color-red)",
                    },
                    {
                      newText: "var(--token-color-blue-lightdark)",
                      range: doc.getRangeForSubstring(
                        "var(--token-color-blue-lightdark, light-dark(lightblue, darkblue))",
                      ),
                    },
                  ],
                },
              },
            },
          ]);
        });
      });
    });
  });
});
