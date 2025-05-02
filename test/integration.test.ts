import { assertEquals } from '@std/assert';

import { createTestLspClient } from '#test-helpers';

import manifest from '../package.json' with { type: 'json' };

const { version } = manifest;

// Test against the running server binary
Deno.test('design-tokens-language-server', async (t) => {
  const client = createTestLspClient();
  await client.sendNotification({ method: 'initialized' });

  try {
    await t.step('initialize', async () => {
      // Step 2: Initialize the LSP server
      const initializeResponse = await client.sendMessage({
        method: 'initialize',
        params: {
          processId: null,
          rootUri: 'file:///test-root/',
          clientInfo: {
            name: 'DENO_TEST_CLIENT',
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

      assertEquals(initializeResponse?.jsonrpc, '2.0');
      assertEquals(initializeResponse?.id, 0);
      assertEquals(initializeResponse?.result.serverInfo.version, version);
      assertEquals(
        initializeResponse?.result.serverInfo.name,
        'design-tokens-language-server',
      );
    });

    await client.sendNotification({ method: 'initialized' });

    const uri = 'file://test.css';

    const initialText = 'body { color: red; }';

    await t.step('didOpen', async () => {
      // Step 3: Open a document
      const didOpenResponse = await client.sendNotification({
        method: 'textDocument/didOpen',
        params: {
          textDocument: {
            uri,
            languageId: 'css',
            version: 1,
            text: initialText,
          },
        },
      });

      assertEquals(didOpenResponse, undefined); // No response expected for didOpen
    });

    await t.step('changes', async () => {
      // Step 4: Simulate incremental document changes
      const change1 = {
        range: {
          start: { line: 0, character: 12 },
          end: { line: 0, character: 15 },
        },
        text: 'blue',
      };
      const change2 = {
        range: {
          start: { line: 0, character: 12 },
          end: { line: 0, character: 16 },
        },
        text: 'green',
      };

      // First incremental update: Change "red" to "blue"
      await client.sendNotification({
        method: 'textDocument/didChange',
        params: {
          textDocument: { uri, version: 2 },
          contentChanges: [change1],
        },
      });

      // Second incremental update: Change "blue" to "green"
      await client.sendNotification({
        method: 'textDocument/didChange',
        params: {
          textDocument: { uri, version: 3 },
          contentChanges: [change2],
        },
      });

      // TODO: add test tokens so that this is meaningful

      // Step 5: Request hover and diagnostics
      const hoverResponse = await client.sendMessage({
        method: 'textDocument/hover',
        params: {
          textDocument: { uri },
          position: { line: 0, character: 10 },
        },
      });

      const diagnosticsResponse = await client.sendMessage({
        method: 'textDocument/diagnostic',
        params: { textDocument: { uri } },
      });

      // Step 6: Assert results
      assertEquals(hoverResponse, { jsonrpc: '2.0', id: 1, result: null });

      assertEquals(diagnosticsResponse, {
        jsonrpc: '2.0',
        id: 2,
        result: { kind: 'full', items: [] },
      }); // Replace with expected diagnostics
    });
  } finally {
    await client.close();
  }
});
