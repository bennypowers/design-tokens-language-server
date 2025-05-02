import type * as LSP from "vscode-languageserver-protocol";
import { beforeEach, describe, it } from "@std/testing/bdd";

import { createTestContext, DTLSTestContext } from "#test-helpers";

import { Workspaces } from "#workspaces";
import { expect } from "@std/expect/expect";

/** a comprehensive test suite for the Documents class */
describe("Documents", () => {
  let ctx: DTLSTestContext;
  let workspaces: Workspaces;

  beforeEach(async () => {
    workspaces = new Workspaces();
    ctx = await createTestContext({ workspaces, testTokensSpecs: [] });
  });

  describe("workspaces/didChangeWorkspaceFolders", () => {
    // TODO: mock the FS
    describe("called with the test package", () => {
      let result: void;
      const workspaceRoot =
        new URL("../../test/package/", import.meta.url).href;
      const params: LSP.DidChangeWorkspaceFoldersParams = {
        event: {
          added: [{ name: "root", uri: workspaceRoot }],
          removed: [],
        },
      };
      beforeEach(async () => {
        const method =
          workspaces.handlers["workspace/didChangeWorkspaceFolders"];
        result = await method(params, ctx);
      });
      it("has no result", () => {
        expect(result).toBeUndefined();
      });
      it("reloads files", () => {
        const refereryaml = new URL("tokens/referer.yaml", workspaceRoot).href;
        workspaces.getSpecForUri(refereryaml);
        expect(workspaces.getPrefixForUri(refereryaml))
          .toEqual("token");
      });
    });
  });

  describe("workspaces/didChangeConfiguration", () => {
    describe("called with some fallback settings", () => {
      let result: void;
      const workspaceRoot =
        new URL("../../test/package/", import.meta.url).href;
      const params: LSP.DidChangeConfigurationParams = {
        settings: { workspaceRoot, prefix: "hooli" },
      };
      beforeEach(async () => {
        const method = workspaces.handlers["workspace/didChangeConfiguration"];
        result = await method(params, ctx);
      });
      it("has no result", () => {
        expect(result).toBeUndefined();
      });
      describe("adding tokens", () => {
        beforeEach(async () => {
          const tokens = {
            color: {
              $type: "color",
              a: { $value: "#aaa" },
            },
          };
          const spec = {
            path: workspaceRoot + "tokens/hooli.json",
          };
          // call didChangeWorkspaceFolders with new tokens - mock this somehow
          workspaces;
        });
        it.only("applies settings to tokens", () => {
          expect(ctx.tokens.get("--hooli-color-a"))
            .toEqual("token");
        });
      });
    });
  });
});
