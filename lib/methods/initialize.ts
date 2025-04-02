import { TextDocumentSyncKind, type InitializeParams, type InitializeResult, type RequestMessage } from 'npm:vscode-languageserver-protocol';

import * as fs from 'jsr:@std/fs';
import { register } from "../storage.ts";
import { Logger } from "../logger.ts";

export interface InitializeRequestMessage extends RequestMessage {
  params: InitializeParams;
};

export async function initialize(message: InitializeRequestMessage): Promise<InitializeResult> {
  for (const { uri } of message.params.workspaceFolders ?? []) {
    const pkgJsonPath = new URL('./package.json', `${uri}/`);
    if (await fs.exists(pkgJsonPath)) {
      const { default: manifest } = await import(pkgJsonPath.href, { with: { type: 'json' } });
      for (const spec of manifest?.designTokensLanguageServer?.tokensFiles ?? []) {
        try {
          await register(spec);
        } catch (e) {
          Deno.stderr.write(new TextEncoder().encode(`${e}`));
        }
      }
    }
  }

  return {
    capabilities: {
      hoverProvider: true,
      textDocumentSync: TextDocumentSyncKind.Full,
      completionProvider: { },
    },
    serverInfo: {
      name: "design-tokens-language-server",
      version: "0.0.1",
    },
  };
}

