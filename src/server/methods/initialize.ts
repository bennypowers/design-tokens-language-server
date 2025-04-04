import {
  InitializeParams,
  InitializeResult,
  TextDocumentSyncKind,
} from "vscode-languageserver-protocol";

import { register } from "../storage.ts";
import { Logger } from "../logger.ts";

export async function initialize(params: InitializeParams): Promise<InitializeResult> {
  for (const { uri } of params.workspaceFolders ?? []) {
    const pkgJsonPath = new URL('./package.json', `${uri}/`);
    const { default: manifest } = await import(pkgJsonPath.href, { with: { type: 'json' } });
    for (const spec of manifest?.designTokensLanguageServer?.tokensFiles ?? [])
      await register(spec)
        .catch(() => Logger.error(`Could not load tokens for ${spec}`));
  }

  return {
    capabilities: {
      hoverProvider: true,
      textDocumentSync: TextDocumentSyncKind.Full,
      // FIXME: completion is totally busted - not clear why
      completionProvider: {
        resolveProvider: true,
        completionItem: {
          labelDetailsSupport: true,
        },
      },
      colorProvider: { },
    },
    serverInfo: {
      name: "design-tokens-language-server",
      version: "0.0.1",
    },
  };
}

