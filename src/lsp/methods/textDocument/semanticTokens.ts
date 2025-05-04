import * as LSP from "vscode-languageserver-protocol";
import { DTLSContext } from "#lsp/lsp.ts";

export async function full(
  params: LSP.SemanticTokensParams,
  context: DTLSContext,
): Promise<LSP.SemanticTokens | null> {
  return null;
}

export async function delta(
  params: LSP.SemanticTokensDeltaParams,
  context: DTLSContext,
): Promise<LSP.SemanticTokensDelta | null> {
  return null;
}

export async function range(
  params: LSP.SemanticTokensRangeParams,
  context: DTLSContext,
): Promise<LSP.SemanticTokens | null> {
  return null;
}

export const capabilities: Partial<LSP.ServerCapabilities> = {
  semanticTokensProvider: {
    legend: {
      tokenTypes: [
        LSP.SemanticTokenTypes.namespace,
        LSP.SemanticTokenTypes.property,
      ],
      tokenModifiers: [],
    },
    full: true,
    // full: { delta: true },
  },
};
