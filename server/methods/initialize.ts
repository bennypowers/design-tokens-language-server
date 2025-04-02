import type { InitializeParams, InitializeResult, TextDocumentSyncKind } from "vscode-languageserver-protocol";

import { register } from "../storage.ts";

export async function initialize(params: InitializeParams): Promise<InitializeResult> {
  for (const { uri } of params.workspaceFolders ?? []) {
    const pkgJsonPath = new URL('./package.json', `${uri}/`);
      try {
        const { default: manifest } = await import(pkgJsonPath.href, { with: { type: 'json' } });
        for (const spec of manifest?.designTokensLanguageServer?.tokensFiles ?? [])
          await register(spec);
      } catch (e) {
        Deno.stderr.write(new TextEncoder().encode(`${e}`));
      }
  }

  return {
    capabilities: {
      hoverProvider: true,
      textDocumentSync: 1 satisfies typeof TextDocumentSyncKind.Full,
      completionProvider: { },
      colorProvider: { },
    },
    serverInfo: {
      name: "design-tokens-language-server",
      version: "0.0.1",
    },
  };
}

