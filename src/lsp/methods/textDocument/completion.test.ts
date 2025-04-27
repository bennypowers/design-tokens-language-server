import { describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { CompletionList, Position } from "vscode-languageserver-protocol";

import { createTestContext } from "#test-helpers";

import { completion } from "./completion.ts";

describe("textDocument/completion", () => {
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

  function getCompletions(content: string, position: Position) {
    const textDocument = ctx.documents.createCssDocument(content);
    return completion({ textDocument, position }, ctx);
  }

  function getCompletionsForWord(word: string, content: string) {
    const textDocument = ctx.documents.createCssDocument(content);
    const doc = ctx.documents.get(textDocument.uri);
    const position = doc.positionForSubstring(word, "end");
    return completion({ textDocument, position }, ctx);
  }

  describe("in an empty document", () => {
    const completions = getCompletions("", { line: 0, character: 0 });
    it("should return no completions", () => {
      expect(completions).toBeNull();
    });
  });

  describe("in a document with a css rule", () => {
    const completions = getCompletionsForWord(
      "a",
      /*css*/ `
      body {
        a`,
    );

    it("should return no completions", () => {
      expect(completions).toBeNull();
    });
  });

  describe("adding the token prefix in a malformed block", () => {
    const completions = getCompletionsForWord(
      "token",
      /*css*/ `
      body {
        token
      }
    `,
    );
    it("should return all token completions", () => {
      expect((completions as CompletionList)?.items).toHaveLength(
        ctx.tokens.size,
      );
    });
  });

  describe("adding the token prefix as a property name", () => {
    const completions = getCompletionsForWord(
      "--token",
      /*css*/ `
      body {
        --token
      }
    `,
    );

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
    const completions = getCompletionsForWord(
      "token",
      /*css*/ `
      body {
        color: token
      }
    `,
    );
    it("should return all token completions", () => {
      expect(completions?.items).toHaveLength(ctx.tokens.size);
    });
    it("should return token completions as var() calls", () => {
      for (const item of completions?.items ?? []) {
        expect(item.textEdit?.newText).toMatch(/^var\(--token/);
      }
    });
  });
});
