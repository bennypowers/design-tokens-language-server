import * as LSP from "vscode-languageserver-protocol";
import { DTLSContext } from "#lsp";

/**
 * Implements the LSP textDocument/definition request for tokens in css or json files.
 *
 * @param params - The LSP definition parameters.
 * @param context - The DTLS context.
 *
 * @returns An array of LSP locations in the token definition JSON file for the token.
 */
export function definition(
  params: LSP.DefinitionParams,
  context: DTLSContext,
): LSP.Location[] {
  return context
    .documents
    .get(params.textDocument.uri)
    .getDefinitions(params, context);
}

export const capabilities: Partial<LSP.ServerCapabilities> = {
  definitionProvider: true,
};
