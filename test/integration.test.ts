import { assertEquals } from "@std/assert";

import { createTestLspClient } from "#test-helpers";

import manifest from "../package.json" with { type: "json" };

const { version } = manifest;

// Test against the running server binary
Deno.test("design-tokens-language-server", async (t) => {
  const client = createTestLspClient();
  await client.sendNotification({ method: "initialized" });

  try {
    await t.step("initialize", async () => {
      // Step 2: Initialize the LSP server
      const rootUri = new URL("../test/package/", import.meta.url).href;
      console.log(rootUri);
      const initializeResponse = await client.sendMessage({
        method: "initialize",
        params: {
          processId: null,
          rootUri,
          workspaceFolders: [
            { uri: rootUri, name: "root" },
          ],
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

      assertEquals(initializeResponse?.jsonrpc, "2.0");
      assertEquals(initializeResponse?.id, 0);
      assertEquals(initializeResponse?.result.serverInfo.version, version);
      assertEquals(
        initializeResponse?.result.serverInfo.name,
        "design-tokens-language-server",
      );
    });

    await client.sendNotification({ method: "initialized" });

    const uri = "file:///test.css";

    const initialText = "body { color: red; }";

    await t.step("didOpen", async () => {
      // Step 3: Open a document
      const didOpenResponse = await client.sendNotification({
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

      assertEquals(didOpenResponse, undefined); // No response expected for didOpen
    });

    await t.step("changes", async () => {
      // Step 4: Simulate incremental document changes
      const change1 = {
        range: {
          start: { line: 0, character: 12 },
          end: { line: 0, character: 15 },
        },
        text: "blue",
      };
      const change2 = {
        range: {
          start: { line: 0, character: 12 },
          end: { line: 0, character: 16 },
        },
        text: "green",
      };

      // First incremental update: Change "red" to "blue"
      await client.sendNotification({
        method: "textDocument/didChange",
        params: {
          textDocument: { uri, version: 2 },
          contentChanges: [change1],
        },
      });

      // Second incremental update: Change "blue" to "green"
      await client.sendNotification({
        method: "textDocument/didChange",
        params: {
          textDocument: { uri, version: 3 },
          contentChanges: [change2],
        },
      });

      // TODO: add test tokens so that this is meaningful

      // Step 5: Request hover and diagnostics
      const hoverResponse = await client.sendMessage({
        method: "textDocument/hover",
        params: {
          textDocument: { uri },
          position: { line: 0, character: 10 },
        },
      });

      const diagnosticsResponse = await client.sendMessage({
        method: "textDocument/diagnostic",
        params: { textDocument: { uri } },
      });

      // Step 6: Assert results
      assertEquals(hoverResponse, { jsonrpc: "2.0", id: 1, result: null });

      assertEquals(diagnosticsResponse, {
        jsonrpc: "2.0",
        id: 2,
        result: { kind: "full", items: [] },
      }); // Replace with expected diagnostics
    });

    const yamlRefererUri = new URL(
      "../test/package/tokens/referer.yaml",
      import.meta.url,
    );

    const yamlContent = await Deno.readTextFile(yamlRefererUri);

    await t.step("open yaml referer", async () => {
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

    await t.step("references from yaml to yaml and json", async () => {
      const position = yamlContent.split("\n").reduce<any>((acc, line, i) => {
        if (acc) {
          return acc;
        } else if (line.includes("{color.red.hex")) {
          return {
            line: i,
            character: line.indexOf("color.") + 1,
          };
        }
      }, undefined);
      const referencesResponse = await client.sendMessage({
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
      assertEquals(referencesResponse, {
        jsonrpc: "2.0",
        id: 3,
        result: [
          {
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
            uri: `file:///${Deno.cwd()}/test/package/tokens/referer.json`,
          },
          {
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
            uri: `file:///${Deno.cwd()}/test/package/tokens/referer.json`,
          },
          {
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
            uri: `file:///${Deno.cwd()}/test/package/tokens/referer.yaml`,
          },
          {
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
            uri: `file:///${Deno.cwd()}/test/package/tokens/referee.json`,
          },
        ],
      });
    });
  } finally {
    await client.close();
  }
});
