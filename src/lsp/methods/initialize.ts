import * as LSP from "vscode-languageserver-protocol";

import { register, tokens } from "#tokens";

import { Logger } from "#logger";

/**
 * The initialize function is called when the server is initialized.
 * It registers the tokens files and sets up the server capabilities.
 *
 * @param params - The parameters for the initialization request.
 * @returns The capabilities of the server.
 */
export async function initialize(params: LSP.InitializeParams): Promise<LSP.InitializeResult> {
  Logger.info`\n\n🎨 DESIGN TOKENS LANGUAGE SERVER 💎: ${params.clientInfo?.name ?? 'unknown-client'}@${params.clientInfo?.version ?? 'unknown-version'}\n`;

  for (const { uri } of params.workspaceFolders ?? []) {
    const pkgJsonPath = new URL('./package.json', `${uri}/`);
    const { default: manifest } = await import(pkgJsonPath.href, { with: { type: 'json' } });
    for (const tokensFile of manifest?.designTokensLanguageServer?.tokensFiles ?? [])
      await register(tokensFile)
        .catch(() => Logger.error`Could not load tokens for ${tokensFile.path}`);
  }

  Logger.info`Available Tokens:\n${Object.fromEntries(
    tokens.entries().filter(([k]) => k.startsWith('--')).map(([k, v]) => [k, v.$value]),
  )}\n`;

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

