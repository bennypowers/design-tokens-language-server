import { describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { CompletionList } from "vscode-languageserver-protocol";

import { createTestContext } from "#test-helpers";

import { completion } from "./completion.ts";

describe("textDocument/completion", () => {
  const ctx = createTestContext();

  describe("in an empty document", () => {
    const textDocument = ctx.documents.create("");

    it("should return no completions", () => {
      const completions = completion({
        textDocument,
        position: { line: 0, character: 0 },
      }, ctx);
      expect(completions).toBeNull();
    });
  });

  describe("in a document with a css rule", () => {
    const textDocument = ctx.documents.create(/*css*/ `
      body {
        a`);

    const position = textDocument.positionOf("a", "end");

    it("should return no completions", () => {
      const completions = completion({ textDocument, position }, ctx);
      expect(completions).toBeNull();
    });
  });

  describe("adding the token prefix in a malformed block", () => {
    const textDocument = ctx.documents.create(/*css*/ `
      body {
        token
      }
    `);
    const position = textDocument.positionOf("token", "end");
    it("should return all token completions", () => {
      const completions = completion({ textDocument, position }, ctx);
      expect((completions as CompletionList)?.items).toHaveLength(
        ctx.tokens.size,
      );
    });
  });

  describe("adding the token prefix as a property name", () => {
    const textDocument = ctx.documents.create(/*css*/ `
      body {
        --token
      }
    `);
    const position = textDocument.positionOf("--token", "end");
    const completions = completion({ textDocument, position }, ctx);

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
    const textDocument = ctx.documents.create(/*css*/ `
      body {
        color: token
      }
    `);
    const position = textDocument.positionOf("token", "end");
    const completions = completion({ textDocument, position }, ctx);
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
