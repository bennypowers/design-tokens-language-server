import { beforeAll, beforeEach, describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { createTestContext } from "#test-helpers";

import { codeAction, DTLSCodeAction } from "./codeAction.ts";

import { resolve } from "../codeAction/resolve.ts";

import { CodeActionKind } from "vscode-languageserver-protocol";
import { DTLSTextDocument } from "#document";

describe("textDocument/codeAction", () => {
  const ctx = createTestContext({
    testTokensSpecs: [
      {
        prefix: "token",
        spec: "file:///tokens.json",
        tokens: {
          color: {
            red: {
              _: {
                $value: "red",
                $type: "color",
              },
            },
            blue: {
              lightdark: {
                $value: "light-dark(lightblue, darkblue)",
                $description: "Color scheme color",
                $type: "color",
              },
            },
          },
        },
      },
    ],
  });

  describe("in a css document with one token var call and no fallback", () => {
    const textDocument = ctx.documents.createCssDocument(/*css*/ `
        a {
          color: var(--token-color-red);
        }
      `);

    const doc = ctx.documents.get(textDocument.uri);

    let result: ReturnType<typeof codeAction>;

    describe("called on the first character of the file", () => {
      beforeEach(() => {
        result = codeAction({
          textDocument,
          range: {
            start: { line: 0, character: 0 },
            end: { line: 0, character: 0 },
          },
          context: {
            diagnostics: doc.diagnostics,
          },
        }, ctx);
      });
      it("should return null", () => {
        expect(result).toBeNull();
      });
    });

    describe("calling codeAction on the first character of the token name", () => {
      beforeEach(() => {
        const position = doc.positionForSubstring("--token-color-red");
        result = codeAction({
          textDocument,
          range: { start: position, end: position },
          context: { diagnostics: doc.diagnostics },
        }, ctx);
      });
      it("should return a single-token refactor action", () => {
        const newText = "var(--token-color-red, red)";
        const range = doc.rangeForSubstring("var(--token-color-red)");
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
    const textDocument = ctx.documents.createCssDocument(/*css*/ `
      a {
        color: var(--token-color-red, blue);
      }
    `);

    const doc = ctx.documents.get(textDocument.uri);

    describe("called on the first character of the fallback", () => {
      let result: ReturnType<typeof codeAction>;

      beforeEach(() => {
        const position = doc.positionForSubstring("blue");
        result = codeAction({
          textDocument,
          range: { start: position, end: position },
          context: { diagnostics: doc.diagnostics },
        }, ctx);
      });

      it("should return a fix for the incorrect fallback", () => {
        expect(result?.at(0)).toEqual({
          title: DTLSCodeAction.fixFallback,
          kind: CodeActionKind.QuickFix,
          diagnostics: doc.diagnostics,
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
    const textDocument = ctx.documents.createCssDocument(/*css*/ `
      a {
        color: var(--token-color-red, red);
      }
    `);

    const doc = ctx.documents.get(textDocument.uri);

    describe("called on the first character of the fallback", () => {
      let result: ReturnType<typeof codeAction>;

      beforeEach(() => {
        const position = doc.positionForSubstring("red");
        result = codeAction({
          textDocument,
          range: { start: position, end: position },
          context: { diagnostics: doc.diagnostics },
        }, ctx);
      });

      it("should return a single refactor action for a token call with fallback", () => {
        const range = doc.rangeForSubstring("var(--token-color-red, red)");
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
    const textDocument = ctx.documents.createCssDocument(/*css*/ `
      a {
        color: var(--token-color-red, blue);
        border-color: var(--token-color-blue-lightdark, green);
      }
    `);

    const doc = ctx.documents.get(textDocument.uri);

    let result: ReturnType<typeof codeAction>;

    beforeEach(() => {
      result = codeAction({
        textDocument,
        range: { start: doc.positionAt(0), end: doc.positionAt(0) },
        context: { diagnostics: doc.diagnostics },
      }, ctx);
    });

    it("should return two fixes for the incorrect fallbacks and a fixall fix", () => {
      expect(result).toEqual([
        {
          title: DTLSCodeAction.fixFallback,
          kind: CodeActionKind.QuickFix,
          data: { textDocument },
          diagnostics: [doc.diagnostics[0]],
        },
        {
          title: DTLSCodeAction.fixFallback,
          kind: CodeActionKind.QuickFix,
          data: { textDocument },
          diagnostics: [doc.diagnostics[1]],
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
      beforeAll(() => {
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
          result = codeAction({
            textDocument: doc.identifier,
            range: {
              start: doc.positionForSubstring("color:", "start"),
              end: doc.positionForSubstring("darkblue));", "end"),
            },
            context: { diagnostics: doc.diagnostics },
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
                      range: doc.rangeForSubstring(
                        "var(--token-color-red, red)",
                      ),
                      newText: "var(--token-color-red)",
                    },
                    {
                      newText: "var(--token-color-blue-lightdark)",
                      range: doc.rangeForSubstring(
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
