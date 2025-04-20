import { describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { cssColorToLspColor } from "#color";

import { TestDocuments, TestTokens } from "#test-helpers";

import { documentColor } from "./documentColor.ts";

describe("documentColor", () => {
  const documents = new TestDocuments();
  const tokens = new TestTokens();
  describe("in a document with a single token with type color", () => {
    const uri = documents.create(`a{b:var(--token-color-red)}\n`, tokens);

    it("should return a single DocumentColor", () => {
      const results = documentColor({ textDocument: { uri } }, {
        documents,
        tokens,
      });
      expect(results).not.toBeNull();
      expect(results).toHaveLength(1);
      const [result] = results;
      expect(result.color).toEqual({
        red: 1,
        green: 0,
        blue: 0,
        alpha: 1,
      });
      expect(result.range).toEqual({
        start: {
          line: 0,
          character: 8,
        },
        end: {
          line: 0,
          character: 25,
        },
      });
    });
  });

  describe("in a document with a single token with type dimension", () => {
    const uri = documents.create(`a{b:var(--token-space-small)}\n`, tokens);

    it("should return an empty array", () => {
      const results = documentColor({ textDocument: { uri } }, {
        documents,
        tokens,
      });
      expect(results).toHaveLength(0);
    });
  });

  describe("in a document with two tokens: one color, one dimension", () => {
    const uri = documents.create(
      `a{b:var(--token-color-red); c:var(--token-space-small)}\n`,
      tokens,
    );

    it("should return a single DocumentColor", () => {
      const results = documentColor({ textDocument: { uri } }, {
        documents,
        tokens,
      });
      expect(results).not.toBeNull();
      expect(results).toHaveLength(1);
      const [result] = results;
      expect(result.color).toEqual(cssColorToLspColor("red"));
      expect(result.range).toEqual({
        start: {
          line: 0,
          character: 8,
        },
        end: {
          line: 0,
          character: 25,
        },
      });
    });
  });

  describe("in a document with a single token with type color and light-dark values", () => {
    const uri = documents.create(
      `a{b:var(--token-color-blue-lightdark)}\n`,
      tokens,
    );

    it("should return two DocumentColors for the same token", () => {
      const results = documentColor({ textDocument: { uri } }, {
        documents,
        tokens,
      });
      expect(results).not.toBeNull();
      expect(results).toHaveLength(2);
      const [result1, result2] = results;
      expect(result1.color).toEqual(cssColorToLspColor("lightblue"));
      expect(result2.color).toEqual(cssColorToLspColor("darkblue"));
      expect(result1.range).toEqual({
        start: {
          line: 0,
          character: 8,
        },
        end: {
          line: 0,
          character: 36,
        },
      });
      expect(result2.range).toEqual(result1.range);
    });
  });
});
