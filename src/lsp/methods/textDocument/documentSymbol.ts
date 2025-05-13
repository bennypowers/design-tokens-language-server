import type * as LSP from "vscode-languageserver-protocol";

import { DTLSContext } from "#lsp";

/**
 * Generates document symbols for a documetn
 *
 * @param params - The parameters for the document symbols request.
 * @param context - The context containing design tokens and documents.
 * @returns An array of document symbols, containing metadata like deprecations
 */
export function documentSymbol(
  params: LSP.DocumentSymbolParams,
  context: DTLSContext,
): LSP.DocumentSymbol[] {
  return context
    .documents
    .get(params.textDocument.uri)
    .getDocumentSymbols(context);
}

export const capabilities: Partial<LSP.ServerCapabilities> = {
  documentSymbolProvider: {
    label: "Design Tokens",
  },
};
