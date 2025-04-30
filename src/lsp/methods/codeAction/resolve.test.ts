import { beforeEach, describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { createTestContext, DTLSTestContext } from "#test-helpers";

import { resolve } from "./resolve.ts";
import { CodeAction } from "vscode-languageserver-protocol";

describe("codeAction/resolve", () => {
  let ctx: DTLSTestContext;

  beforeEach(async () => {
    ctx = await createTestContext({
      // NOTE: truthy path is tested in codeAction.test.ts
      testTokensSpecs: [],
    });
  });

  describe("given a code action that somehow represents a non-token", () => {
    const codeAction: CodeAction = {
      title: "Test Code Action",
      kind: "refactor",
      diagnostics: [],
      edit: {
        changes: {
          "file:///test.css": [
            {
              range: {
                start: { line: 0, character: 0 },
                end: { line: 0, character: 0 },
              },
              newText: "--token-bogus",
            },
          ],
        },
      },
    };
    it("should resolve the unchanged code action", () => {
      const resolvedItem = resolve(codeAction, ctx);
      expect(resolvedItem).toEqual(codeAction);
    });
  });
});
