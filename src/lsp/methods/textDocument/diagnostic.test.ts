import { describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { createTestContext } from "#test-helpers";

import { diagnostic } from "./diagnostic.ts";

describe("diagnostic", () => {
  const ctx = createTestContext();

  describe("in a document with a single token and no fallback", () => {
    const textDocument = ctx.documents.create(/*css*/ `
      body { color: var(--token-color); }
    `);

    it("should return no diagnostics", () => {
      const diagnostics = diagnostic({ textDocument }, ctx);
      if (diagnostics.kind !== "full") {
        throw new Error("Expected full diagnostics");
      }
      expect(diagnostics.items).toHaveLength(0);
    });
  });

  describe("in a document with a single token and an incorrect fallback", () => {
    const textDocument = ctx.documents.create(/*css*/ `
      body {
        color: var(--token-color-red, blue);
      }
    `);

    it("should return a single diagnostic", () => {
      const diagnostics = diagnostic({ textDocument }, ctx);
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
