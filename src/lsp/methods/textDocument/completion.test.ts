import { beforeEach, describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { TestDocuments, TestTokens } from "#test-helpers";
import { CompletionList } from "vscode-languageserver-protocol";

import { completion } from "./completion.ts";

describe("completion", () => {
  const documents = new TestDocuments();
  const tokens = new TestTokens();

  describe("in an empty document", () => {
    const uri = documents.create("", tokens);

    it("should return no completions", () => {
      const completions = completion({
        textDocument: { uri },
        position: { line: 0, character: 0 },
      }, { documents, tokens });
      expect(completions).toBeNull();
    });
  });

  describe("in a document with a css rule", () => {
    const uri = documents.create("body {\n  ", tokens);

    it("should return no completions", () => {
      const completions = completion({
        textDocument: { uri },
        position: { line: 1, character: 3 },
      }, { documents, tokens });
      expect(completions).toBeNull();
    });
  });

  describe("adding the token prefix in a malformed block", () => {
    const uri = documents.create("body {\n  token }", tokens);
    it("should return all token completions", () => {
      const completions = completion({
        textDocument: { uri },
        position: { line: 1, character: 5 },
      }, { documents, tokens });
      expect((completions as CompletionList)?.items).toHaveLength(8);
    });
  });

  describe("adding the token prefix as a property name", () => {
    const uri = documents.create("body {\n  --token }", tokens);
    let completions: CompletionList | null;
    beforeEach(() => {
      completions = completion({
        textDocument: { uri },
        position: { line: 1, character: 8 },
      }, { documents, tokens }) as CompletionList;
    });
    it("should return all token completions", () => {
      expect(completions?.items).toHaveLength(8);
    });
    it("should return token completions as property names", () => {
      for (const item of completions?.items ?? []) {
        expect(item.textEdit?.newText).toMatch(/^--token/);
      }
    });
  });

  describe("adding the token prefix as a property value", () => {
    const uri = documents.create("body {\n  color: token }", tokens);
    let completions: CompletionList | null;
    beforeEach(() => {
      completions = completion({
        textDocument: { uri },
        position: { line: 1, character: 14 },
      }, { documents, tokens }) as CompletionList;
    });
    it("should return all token completions", () => {
      expect(completions?.items).toHaveLength(8);
    });
    it("should return token completions as var() calls", () => {
      for (const item of completions?.items ?? []) {
        expect(item.textEdit?.newText).toMatch(/^var\(--token/);
      }
    });
  });
});
