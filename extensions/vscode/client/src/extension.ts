import * as path from "node:path";
import { ExtensionContext } from "vscode";

import { LanguageClient, TransportKind } from "vscode-languageclient/node";

let client: LanguageClient;

export function activate(context: ExtensionContext) {
  const command = context.asAbsolutePath(path.join('dist', 'bin', 'design-tokens-language-server'));
  const args: string[] = [];

  // const command = path.join(context.extensionPath, 'dist', 'bin', 'deno');
  // const args = [
  //   'run',
  //   '--unstable-temporal',
  //   '-A',
  //   path.join(context.extensionPath, 'dist', 'main.js'),
  //   '--stdio'
  // ];

  client = new LanguageClient(
    "design-tokens-language-server",
    "Design Tokens Language Server",
    {
      run: { command, args, transport: TransportKind.stdio },
      debug: { command, args, transport: TransportKind.stdio },
    },
    {
      documentSelector: [
        { scheme: "file", language: "css" },
      ],
    },
  );

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
