import { describe, it, afterAll, beforeAll } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { createTestLspClient } from "#test-helpers";

import manifest from "../package.json" with { type: "json" };

const { version } = manifest;

describe("design-tokens-language-server", () => {
  let client: ReturnType<typeof createTestLspClient>;

  beforeAll(async () => {
    client = createTestLspClient();
    await client.sendNotification({ method: "initialized" });
  });

  afterAll(async () => {
    await client.close();
  });

  it("should initialize the LSP server", async () => {
    const rootUri = new URL("../test/package/", import.meta.url).href;
    const initializeResponse = await client.sendMessage({
      method: "initialize",
      params: {
        processId: null,
        rootUri,
        workspaceFolders: [{ uri: rootUri, name: "root" }],
        clientInfo: {
          name: "DENO_TEST_CLIENT",
          version: Temporal.Now.plainDateTimeISO().toString(),
        },
        capabilities: {
          textDocument: {
            synchronization: {
              dynamicRegistration: false,
              willSave: false,
              didSave: false,
              willSaveWaitUntil: false,
            },
          },
        },
      },
    });

    expect(initializeResponse?.jsonrpc).toBe("2.0");
    expect(initializeResponse?.id).toBe(0);
    expect(initializeResponse?.result.serverInfo.version).toBe(version);
    expect(initializeResponse?.result.serverInfo.name).toBe(
      "design-tokens-language-server",
    );
  });

  describe("calling didOpen on a test file", () => {
    const uri = "file:///test.css";
    const initialText = "body { color: red; }";
    let didOpenResponse: any;
    beforeAll(async () => {
      didOpenResponse = await client.sendNotification({
        method: "textDocument/didOpen",
        params: {
          textDocument: {
            uri,
            languageId: "css",
            version: 1,
            text: initialText,
          },
        },
      });
    });

    it("should not respond to the didOpen notification", () => {
      // Step 3: Open a document
      expect(didOpenResponse).toBeUndefined(); // No response expected for didOpen
    });

    describe("calling didChange", () => {
      // Step 4: Simulate incremental document changes
      beforeAll(async () => {
        // First incremental update: Change "red" to "blue"
        await client.sendNotification({
          method: "textDocument/didChange",
          params: {
            textDocument: { uri, version: 2 },
            contentChanges: [
              {
                range: {
                  start: { line: 0, character: 12 },
                  end: { line: 0, character: 15 },
                },
                text: "blue",
              },
            ],
          },
        });

        // Second incremental update: Change "blue" to "green"
        await client.sendNotification({
          method: "textDocument/didChange",
          params: {
            textDocument: { uri, version: 3 },
            contentChanges: [
              {
                range: {
                  start: { line: 0, character: 12 },
                  end: { line: 0, character: 16 },
                },
                text: "green",
              },
            ],
          },
        });
      });

      describe("then calling hover on a non-token", () => {
        let hoverResponse: any;
        beforeAll(async () => {
          // Step 5: Request hover and diagnostics
          hoverResponse = await client.sendMessage({
            method: "textDocument/hover",
            params: {
              textDocument: { uri },
              position: { line: 0, character: 10 },
            },
          });
        });

        it("should return null hover information", () => {
          // Step 6: Assert results
          expect(hoverResponse).toEqual({
            jsonrpc: "2.0",
            id: 1,
            result: null,
          });
        });
        describe("then calling diagnostic", () => {
          let diagnosticsResponse: any;
          beforeAll(async () => {
            diagnosticsResponse = await client.sendMessage({
              method: "textDocument/diagnostic",
              params: { textDocument: { uri } },
            });

            expect(diagnosticsResponse).toEqual({
              jsonrpc: "2.0",
              id: 2,
              result: { kind: "full", items: [] },
            }); // Replace with expected diagnostics
          });
        });
      });
    });
  });

  describe("given a yaml file", () => {
    let yamlContent: string;
    const yamlRefererUri = new URL(
      "../test/package/tokens/referer.yaml",
      import.meta.url,
    );

    beforeAll(async () => {
      yamlContent = await Deno.readTextFile(yamlRefererUri);

      // Step 7: Open YAML referer
      await client.sendNotification({
        method: "textDocument/didOpen",
        params: {
          textDocument: {
            uri: yamlRefererUri.href,
            languageId: "yaml",
            version: 1,
            text: yamlContent,
          },
        },
      });
    });

    describe("then calling references", () => {
      let referencesResponse: any;
      let position: any;

      beforeAll(async () => {
        // Step 8: References from YAML to YAML and JSON
        position = yamlContent.split("\n").reduce<any>((acc, line, i) => {
          if (acc) {
            return acc;
          } else if (line.includes("{color.red.hex")) {
            return {
              line: i,
              character: line.indexOf("color.") + 1,
            };
          }
        }, undefined);

        referencesResponse = await client.sendMessage({
          method: "textDocument/references",
          params: {
            textDocument: {
              uri: yamlRefererUri.href,
            },
            context: {
              includeDeclaration: true,
            },
            position,
          },
        });
      });

      it("gathers the correct references", () => {
        // TODO: get context from the test client and compute these values
        expect(referencesResponse).toEqual({
          jsonrpc: "2.0",
          id: 3,
          result: [
            {
              uri: `file://${Deno.cwd()}/test/package/tokens/referer.json`,
              range: {
                end: {
                  character: 34,
                  line: 5,
                },
                start: {
                  character: 19,
                  line: 5,
                },
              },
            },
            {
              uri: `file://${Deno.cwd()}/test/package/tokens/referer.json`,
              range: {
                end: {
                  character: 45,
                  line: 17,
                },
                start: {
                  character: 30,
                  line: 17,
                },
              },
            },
            {
              uri: `file://${Deno.cwd()}/test/package/tokens/referer.yaml`,
              range: {
                end: {
                  character: 28,
                  line: 7,
                },
                start: {
                  character: 13,
                  line: 7,
                },
              },
            },
            {
              uri: `file://${Deno.cwd()}/test/package/tokens/referer.yaml`,
              range: {
                end: {
                  character: 55,
                  line: 10,
                },
                start: {
                  character: 40,
                  line: 10,
                },
              },
            },
            {
              uri: `file://${Deno.cwd()}/test/package/tokens/referee.json`,
              range: {
                end: {
                  character: 7,
                  line: 42,
                },
                start: {
                  character: 13,
                  line: 39,
                },
              },
            },
          ],
        });
      });
    });
  });
});
