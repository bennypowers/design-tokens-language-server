import path = require("node:path");
import { ExtensionContext, workspace } from "vscode";

import { LanguageClient, TransportKind } from "vscode-languageclient/node";

let client: LanguageClient;

export function activate(context: ExtensionContext) {
  const command = context.asAbsolutePath(path.join('bin', 'design-tokens-language-server'));
  // Create the language client and start the client.
  client = new LanguageClient(
    "design-tokens-language-server",
    "Design Tokens Language Server",
    // If the extension is launched in debug mode then the debug server options are used
    // Otherwise the run options are used
    {
      run: { command, transport: TransportKind.stdio },
      debug: { command, transport: TransportKind.stdio },
    },
    // Options to control the language client
    {
      // Register the server for all documents by default
      documentSelector: [
        { scheme: "file", language: "css" },
        // TODO: vscode will open the package.json for a project folder with textDocument/didOpen, and that will not parse in #handleChunk
        // { scheme: "file", language: "json" },
        // { scheme: "file", language: "yaml" },
      ],
      synchronize: {
        // Notify the server about file changes to '.clientrc files contained in the workspace
        fileEvents: workspace.createFileSystemWatcher("**/.clientrc"),
      },
    },
  );

  // Start the client. This will also launch the server
  client.start().then(() => {
    console.log("STARTED")
  });
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}
