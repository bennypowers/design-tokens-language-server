import {
  afterAll,
  beforeAll,
  beforeEach,
  describe,
  it,
} from "@std/testing/bdd";
import { expect } from "@std/expect";

import { register, tokens } from "#tokens";
import { completion } from "./completion.ts";
import { TestDocuments } from "#test-helpers";
import { CompletionList } from "vscode-languageserver-protocol";

describe("completion", () => {
  const documents = new TestDocuments();

  beforeAll(async () => {
    await register({ path: "./test/tokens.json", prefix: "token" });
  });

  afterAll(() => {
    documents.tearDown();
    tokens.clear();
  });

  describe("in an empty document", () => {
    const uri = documents.create("");

    it("should return no completions", () => {
      const completions = completion({
        textDocument: { uri },
        position: { line: 0, character: 0 },
      }, documents);
      expect(completions).toBeNull();
    });
  });

  describe("in a document with a css rule", () => {
    const uri = documents.create("body {\n  ");

    it("should return no completions", () => {
      const completions = completion({
        textDocument: { uri },
        position: { line: 1, character: 3 },
      }, documents);
      expect(completions).toBeNull();
    });
  });

  describe("adding the token prefix in a malformed block", () => {
    const uri = documents.create("body {\n  token }");
    it("should return all token completions", () => {
      const completions = completion({
        textDocument: { uri },
        position: { line: 1, character: 5 },
      }, documents);
      expect((completions as CompletionList)?.items).toHaveLength(8);
    });
  });

  describe("adding the token prefix as a property name", () => {
    const uri = documents.create("body {\n  --token }");
    let completions: CompletionList | null;
    beforeEach(() => {
      completions = completion({
        textDocument: { uri },
        position: { line: 1, character: 8 },
      }, documents) as CompletionList;
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
    const uri = documents.create("body {\n  color: token }");
    let completions: CompletionList | null;
    beforeEach(() => {
      completions = completion({
        textDocument: { uri },
        position: { line: 1, character: 14 },
      }, documents) as CompletionList;
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
