import { describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { TestDocuments, TestTokens } from "#test-helpers";

import { diagnostic } from "./diagnostic.ts";

describe("diagnostic", () => {
  const documents = new TestDocuments();
  const tokens = new TestTokens();

  describe("in a document with a single token and no fallback", () => {
    const uri = documents.create(`body { color: var(--token-color); }`, tokens);

    it("should return no diagnostics", () => {
      const diagnostics = diagnostic({ textDocument: { uri } }, {
        documents,
        tokens,
      });
      if (diagnostics.kind !== "full") {
        throw new Error("Expected full diagnostics");
      }
      expect(diagnostics.items).toHaveLength(0);
    });
  });

  describe("in a document with a single token and an incorrect fallback", () => {
    const uri = documents.create(
      `body { color: var(--token-color-red, blue); }`,
      tokens,
    );

    it("should return a single diagnostic", () => {
      const diagnostics = diagnostic({ textDocument: { uri } }, {
        documents,
        tokens,
      });
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
