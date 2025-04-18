import { assertEquals } from "jsr:@std/assert";
import { LspClient } from "./LspClient.ts";

// Test against the running server binary
Deno.test("should handle rapid document changes without race conditions", async (t) => {
  // Step 1: Start the language server binary
  const server = new Deno.Command(Deno.execPath(), {
    stdin: "piped",
    stdout: "piped",
    stderr: "piped",
    args: ["-A", "--quiet", "./src/main.ts"],
  }).spawn();

  const client = new LspClient(server);

  try {
    await t.step("initialize", async () => {
      // Step 2: Initialize the LSP server
      const initializeResponse = await client.sendLspMessage({
        method: "initialize",
        params: {
          processId: null,
          rootUri: "file:///",
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

      await client.sendNotification({ method: "initialized" });

      assertEquals(initializeResponse, {
        jsonrpc: "2.0",
        id: 1,
        result: {
          capabilities: {
            codeActionProvider: {
              codeActionKinds: [
                "quickfix",
                "refactor.rewrite",
                "source.fixAll",
              ],
              resolveProvider: true,
            },
            colorProvider: true,
            completionProvider: {
              completionItem: {
                labelDetailsSupport: true,
              },
              resolveProvider: true,
            },
            diagnosticProvider: {
              interFileDependencies: false,
              workspaceDiagnostics: false,
            },
            hoverProvider: true,
            textDocumentSync: 2,
          },
          serverInfo: {
            name: "design-tokens-language-server",
            version: "0.0.1",
          },
        },
      });
    });

    const uri = "file://test.css";

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
      const hoverResponse = await client.sendLspMessage({
        method: "textDocument/hover",
        params: {
          textDocument: { uri },
          position: { line: 0, character: 10 },
        },
      });

      const diagnosticsResponse = await client.sendLspMessage({
        method: "textDocument/diagnostic",
        params: { textDocument: { uri } },
      });

      // Step 6: Assert results
      assertEquals(hoverResponse, { jsonrpc: "2.0", id: 2, result: null });

      assertEquals(diagnosticsResponse, {
        jsonrpc: "2.0",
        id: 3,
        result: { kind: "full", items: [] },
      }); // Replace with expected diagnostics
    });
  } finally {
    await client.close();
  }
});
