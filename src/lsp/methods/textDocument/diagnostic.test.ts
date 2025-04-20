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
import { diagnostic } from "./diagnostic.ts";

describe("diagnostic", () => {
  beforeAll(async () => {
    await register({ path: "./test/tokens.json", prefix: "token" });
  });

  afterAll(() => {
    tokens.clear();
  });

  describe("in a document with a single token and no fallback", () => {
    const uri = "test.css";
    beforeEach(() => {
      documents.onDidOpen({
        textDocument: {
          uri,
          languageId: "css",
          version: 1,
          text: `body { color: var(--token-color); }`,
        },
      });
    });

    afterEach(() => {
      documents.onDidClose({ textDocument: { uri } });
    });

    it("should return no diagnostics", () => {
      const diagnostics = diagnostic({ textDocument: { uri } });
      if (diagnostics.kind !== "full") {
        throw new Error("Expected full diagnostics");
      }
      expect(diagnostics.items).toHaveLength(0);
    });
  });

  describe("in a document with a single token and an incorrect fallback", () => {
    const uri = "test-incorrect-fallback.css";
    beforeEach(() => {
      documents.onDidOpen({
        textDocument: {
          uri,
          languageId: "css",
          version: 1,
          text: `body { color: var(--token-color-red, blue); }`,
        },
      });
    });

    afterEach(() => {
      documents.onDidClose({ textDocument: { uri } });
    });

    it("should return a single diagnostic", () => {
      const diagnostics = diagnostic({ textDocument: { uri } });
      if (diagnostics.kind !== "full") {
        throw new Error("Expected full diagnostics");
      }
      expect(diagnostics.items).toHaveLength(1);
      const [diag] = diagnostics.items;
      expect(diag.message).toBe(
        "Token fallback does not match expected value: red",
      );
    });
  });
});
