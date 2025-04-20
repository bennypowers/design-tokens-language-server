import * as LSP from "vscode-languageserver-protocol";

import { register, tokens } from "#tokens";
import { Logger } from "#logger";

export type SupportedNotificationMessage = LSP.NotificationMessage & (
| { method: 'textDocument/didOpen'; params: LSP.DidOpenTextDocumentParams }
| { method: 'textDocument/didChange'; params: LSP.DidChangeTextDocumentParams }
| { method: 'textDocument/didClose'; params: LSP.DidCloseTextDocumentParams }
| { method: '$/setTrace'; params: LSP.SetTraceParams }
| { method: '$/cancelRequest'; params: LSP.RequestMessage }
);

export type SupportedRequestMessage = LSP.RequestMessage & (
| { method: 'initialize'; params: LSP.InitializeParams }
| { method: 'initialized'; params: LSP.InitializedParams }
| { method: 'textDocument/diagnostic'; params: LSP.DocumentDiagnosticParams }
| { method: 'textDocument/documentColor'; params: LSP.DocumentColorParams }
| { method: 'textDocument/hover'; params: LSP.HoverParams }
| { method: 'textDocument/completion'; params: LSP.CompletionParams }
| { method: 'textDocument/codeAction'; params: LSP.CodeActionParams }
| { method: 'codeAction/resolve'; params: LSP.CodeAction }
| { method: 'completionItem/resolve'; params: LSP.CompletionItem }
);

export type SupportedMessage = SupportedRequestMessage | SupportedNotificationMessage;


export async function initialize(params: LSP.InitializeParams): Promise<LSP.InitializeResult> {
  Logger.info`\n\n🎨 DESIGN TOKENS LANGUAGE SERVER 💎: ${params.clientInfo?.name ?? 'unknown-client'}@${params.clientInfo?.version ?? 'unknown-version'}\n`;

  for (const { uri } of params.workspaceFolders ?? []) {
    const pkgJsonPath = new URL('./package.json', `${uri}/`);
    const { default: manifest } = await import(pkgJsonPath.href, { with: { type: 'json' } });
    for (const tokensFile of manifest?.designTokensLanguageServer?.tokensFiles ?? [])
      await register(tokensFile)
        .catch(() => Logger.error`Could not load tokens for ${tokensFile.path}`);
  }

  Logger.info`Available Tokens:\n${Object.fromEntries(tokens.entries().filter(([k]) => k.startsWith('--')))}\n`;

  return {
    capabilities: {
      colorProvider: true,
      hoverProvider: true,
      textDocumentSync: LSP.TextDocumentSyncKind.Incremental,
      completionProvider: {
        resolveProvider: true,
        completionItem: {
          labelDetailsSupport: true,
        },
      },
      codeActionProvider: {
        codeActionKinds: [
          LSP.CodeActionKind.QuickFix,
          LSP.CodeActionKind.RefactorRewrite,
          LSP.CodeActionKind.SourceFixAll,
        ],
        resolveProvider: true,
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

