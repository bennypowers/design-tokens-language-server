import {
  afterAll,
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  it,
} from "@std/testing/bdd";
import { expect } from "@std/expect";

import { register, tokens } from "#tokens";
import { documents } from "#css";
import { documentColor } from "./documentColor.ts";
import { cssColorToLspColor } from "../../../css/color.ts";

describe("documentColor", () => {
  beforeAll(async () => {
    await register({ path: "./test/tokens.json", prefix: "token" });
  });

  afterAll(() => {
    tokens.clear();
  });

  describe("in a document with a single token with type color", () => {
    const uri = "file:///single-color-token.css";
    beforeEach(() => {
      documents.onDidOpen({
        textDocument: {
          uri,
          languageId: "css",
          version: 0,
          text: `a{b:var(--token-color-red)}\n`,
        },
      });
    });

    afterEach(() => {
      documents.onDidClose({ textDocument: { uri } });
    });

    it("should return a single DocumentColor", () => {
      const results = documentColor({ textDocument: { uri } });
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
    const uri = "file:///single-dimension-token.css";
    beforeEach(() => {
      documents.onDidOpen({
        textDocument: {
          uri,
          languageId: "css",
          version: 0,
          text: `a{b:var(--token-space-small)}\n`,
        },
      });
    });

    afterEach(() => {
      documents.onDidClose({ textDocument: { uri } });
    });

    it("should return an empty array", () => {
      const results = documentColor({ textDocument: { uri } });
      expect(results).toHaveLength(0);
    });
  });

  describe("in a document with two tokens: one color, one dimension", () => {
    const uri = "file:///two-tokens-one-color-one-space.css";
    beforeEach(() => {
      documents.onDidOpen({
        textDocument: {
          uri,
          languageId: "css",
          version: 0,
          text: `a{b:var(--token-color-red); c:var(--token-space-small)}\n`,
        },
      });
    });

    afterEach(() => {
      documents.onDidClose({ textDocument: { uri } });
    });

    it("should return a single DocumentColor", () => {
      const results = documentColor({ textDocument: { uri } });
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
    const uri = "file:///single-color-token-light-dark.css";
    beforeEach(() => {
      documents.onDidOpen({
        textDocument: {
          uri,
          languageId: "css",
          version: 0,
          text: `a{b:var(--token-color-blue-lightdark)}\n`,
        },
      });
    });

    afterEach(() => {
      documents.onDidClose({ textDocument: { uri } });
    });

    it("should return two DocumentColors for the same token", () => {
      const results = documentColor({ textDocument: { uri } });
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
