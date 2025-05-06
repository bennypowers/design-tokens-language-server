import * as LSP from "vscode-languageserver-protocol";
import { DTLSContext } from "#lsp";
import { Logger } from "#logger";

export interface DTLSSemanticTokenIntermediate {
  line: number;
  startChar: number;
  length: number;
  tokenType: LSP.SemanticTokenTypes;
  tokenModifiers: number;
  token?: string;
}

export const DTLSTokenTypes = [
  LSP.SemanticTokenTypes.class,
  LSP.SemanticTokenTypes.property,
];

export function full(
  params: LSP.SemanticTokensParams,
  context: DTLSContext,
): LSP.SemanticTokens | null {
  const doc = context.documents.get(params.textDocument.uri);
  switch (doc.language) {
    case "yaml":
    case "json":
      return {
        data: doc.getSemanticTokensFull().flatMap((intermediate, i, a) => {
          Logger.debug`${i}: ${intermediate}`;
          const {
            line,
            startChar,
            length,
            tokenType,
            tokenModifiers,
          } = intermediate;
          const prev = a[i - 1] ?? { line: 0, startChar: 0 };
          const deltaLine = line - prev.line;
          const deltaStart = deltaLine === 0
            ? startChar - prev.startChar
            : startChar;
          return [
            deltaLine,
            deltaStart,
            length,
            DTLSTokenTypes.indexOf(tokenType),
            tokenModifiers,
          ];
        }),
      };
    default:
      return null;
  }
}

/*
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
*/

export const capabilities: Partial<LSP.ServerCapabilities> = {
  semanticTokensProvider: {
    legend: {
      tokenTypes: DTLSTokenTypes,
      tokenModifiers: [],
    },
    full: true,
    // full: { delta: true },
  },
};
