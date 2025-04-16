import {
  InitializeParams,
  InitializeResult,
  TextDocumentSyncKind,
  CodeActionKind,
} from "vscode-languageserver-protocol";

import { register } from "../storage.ts";
import { Logger } from "../logger.ts";

export async function initialize(params: InitializeParams): Promise<InitializeResult> {
  for (const { uri } of params.workspaceFolders ?? []) {
    const pkgJsonPath = new URL('./package.json', `${uri}/`);
    const { default: manifest } = await import(pkgJsonPath.href, { with: { type: 'json' } });
    for (const tokensFile of manifest?.designTokensLanguageServer?.tokensFiles ?? [])
      await register(tokensFile)
        .catch(() => Logger.error(`Could not load tokens for {path}`, { path: tokensFile.path}));
  }

  return {
    capabilities: {
      colorProvider: true,
      hoverProvider: true,
      textDocumentSync: TextDocumentSyncKind.Full,
      // FIXME: completion is totally busted - not clear why
      completionProvider: {
        resolveProvider: true,
        completionItem: {
          labelDetailsSupport: true,
        },
      },
      codeActionProvider: {
        codeActionKinds: [
          CodeActionKind.QuickFix,
          CodeActionKind.RefactorRewrite,
          CodeActionKind.SourceFixAll,
        ]
      },
      diagnosticProvider: {
        interFileDependencies: false,
        workspaceDiagnostics: false,
      }
    },
    serverInfo: {
      name: "design-tokens-language-server",
      version: "0.0.1",
    },
  };
}

