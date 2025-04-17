import { assertEquals } from "jsr:@std/assert";
import { TextDocumentSyncKind } from "npm:vscode-languageserver-protocol";

// Utility function to send and receive LSP messages
async function sendLspMessage(
  process: Deno.ChildProcess,
  message: Record<string, unknown>,
): Promise<Record<string, unknown>> {
  const jsonMessage = JSON.stringify(message);
  const contentLength = `Content-Length: ${jsonMessage.length}\r\n\r\n`;
  const fullMessage = contentLength + jsonMessage;

  // Write the LSP message to the server's stdin
  await process.stdin.getWriter().write(new TextEncoder().encode(fullMessage));

  // Read the response from the server's stdout
  const buffer = await process.stdout.getReader().read();

  const response = new TextDecoder().decode(buffer.value);

  const jsonResponseStart = response.indexOf("{");
  const jsonResponse = response.slice(jsonResponseStart).trim();

  console.log(response)
  return JSON.parse(jsonResponse);
}

// Test against the running server binary
Deno.test("should handle rapid document changes without race conditions", async () => {
  // Step 1: Start the language server binary
  const server = new Deno.Command(Deno.execPath(), {
    stdin: 'piped',
    stdout: 'piped',
    args: ["-A", "--quiet", "./src/main.ts"],
  }).spawn();

  try {
    // Step 2: Initialize the LSP server
    const initializeResponse = await sendLspMessage(server, {
      jsonrpc: "2.0",
      id: 1,
      method: "initialize",
      params: {
        processId: null,
        rootUri: "file:///",
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

    assertEquals((initializeResponse.result as any)?.capabilities.textDocumentSync, TextDocumentSyncKind.Full);

    // Step 3: Open a document
    const uri = "file://test.css";
    const initialText = "body { color: red; }";
    const didOpenResponse = await sendLspMessage(server, {
      jsonrpc: "2.0",
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

    // Step 4: Simulate rapid document changes
    const updatedText1 = "body { color: blue; }";
    const updatedText2 = "body { color: green; }";

    await sendLspMessage(server, {
      jsonrpc: "2.0",
      method: "textDocument/didChange",
      params: {
        textDocument: { uri, version: 2 },
        contentChanges: [{ text: updatedText1 }],
      },
    });

    await sendLspMessage(server, {
      jsonrpc: "2.0",
      method: "textDocument/didChange",
      params: {
        textDocument: { uri, version: 3 },
        contentChanges: [{ text: updatedText2 }],
      },
    });

    // Step 5: Request hover and diagnostics
    const hoverResponse = await sendLspMessage(server, {
      jsonrpc: "2.0",
      method: "textDocument/hover",
      params: {
        textDocument: { uri },
        position: { line: 0, character: 10 },
      },
    });

    const diagnosticsResponse = await sendLspMessage(server, {
      jsonrpc: "2.0",
      method: "textDocument/publishDiagnostics",
      params: { uri },
    });

    // Step 6: Assert results
    assertEquals((hoverResponse.result as any)?.contents, "Expected hover content"); // Replace with expected hover content
    assertEquals((diagnosticsResponse.result as any)?.diagnostics, []); // Replace with expected diagnostics
  } finally {
    server.kill();
  }
});
