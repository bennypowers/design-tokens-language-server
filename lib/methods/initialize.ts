import { TextDocumentSyncKind, type InitializeParams, type InitializeResult, type RequestMessage } from 'npm:vscode-languageserver-protocol';

import * as fs from 'jsr:@std/fs';
import { register } from "../storage.ts";

/*
error: Uncaught (in promise) TypeError: Could not resolve 'npm:@rhds/tokens@3.0.0/json/rhds.tokens.json'
Caused by:
[ERR_INVALID_PACKAGE_TARGET] Invalid \"exports\" target {\"require\":\"./json/*\"} defined for './json/*' in the package config /var/home/bennyp/.cache/deno/npm/registry.npmjs.org/@rhds/tokens/3.0.0/package.json imported from 'file:///var/home/bennyp/Developer/design-tokens-languageserver/storage.ts'; target must start with \"./\"
const { default: json } = await import(spec, { with: { type: 'json'} });
^
at async register (file:///var/home/bennyp/Developer/design-tokens-languageserver/storage.ts:7:29)
at async initialize (file:///var/home/bennyp/Developer/design-tokens-languageserver/methods/initialize.ts:23:9)
at async Server.#handleChunk (file:///var/home/bennyp/Developer/design-tokens-languageserver/server.ts:44:22)
*/

export interface InitializeRequestMessage extends RequestMessage {
  params: InitializeParams;
};

export async function initialize(message: InitializeRequestMessage): Promise<InitializeResult> {
  let completionProvider = undefined;
  for (const { uri } of message.params.workspaceFolders ?? []) {
    const pkgJsonPath = new URL('./package.json', `${uri}/`);
    if (await fs.exists(pkgJsonPath)) {
      const { default: manifest } = await import(pkgJsonPath.href, { with: { type: 'json' } });
      for (const spec of manifest?.designTokensLanguageServer?.tokensFiles ?? []) {
        completionProvider ??= {};
        await register(spec);
      }
    }
  }

  return {
    capabilities: {
      completionProvider,
      hoverProvider: true,
      textDocumentSync: TextDocumentSyncKind.Full,
    },
    serverInfo: {
      name: "design-tokens-languageserver",
      version: "0.0.1",
    },
  };
}

