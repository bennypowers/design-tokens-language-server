import * as path from "node:path";
import { ExtensionContext } from "vscode";

import {
  LanguageClient,
  LanguageClientOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient;

export async function activate(context: ExtensionContext) {
  const command = context.asAbsolutePath(
    path.join("dist", "bin", "design-tokens-language-server"),
  );
  const args: string[] = [];

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "css" }],
  };

  client = new LanguageClient(
    "design-tokens-language-server",
    "Design Tokens Language Server",
    {
      run: { command, args, transport: TransportKind.stdio },
      debug: { command, args, transport: TransportKind.stdio },
    },
    clientOptions,
  );

  try {
    await client.start();
  } catch (error) {
    console.error(error);
  }
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}
