import * as LSP from "vscode-languageserver-protocol";

import { Logger } from "#logger";
import { DTLSContextWithLsp } from "#lsp";

/**
 * The initialize function is called when the server is initialized.
 * It registers the tokens files and sets up the server capabilities.
 *
 * @param params - The parameters for the initialization request.
 * @returns The capabilities of the server.
 */
export async function initialize(
  params: LSP.InitializeParams,
  context: DTLSContextWithLsp,
): Promise<LSP.InitializeResult> {
  Logger.info`\n\n🎨 DESIGN TOKENS LANGUAGE SERVER 💎: ${
    params.clientInfo?.name ?? "unknown-client"
  }@${params.clientInfo?.version ?? "unknown-version"}\n`;

  try {
    await context.lsp.initialize(params, context);
  } catch (error) {
    Logger.error`Failed to initialize the server: ${error}`;
  }

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
      },
    },
    serverInfo: {
      name: "design-tokens-language-server",
      version: "0.0.1",
    },
  };
}
