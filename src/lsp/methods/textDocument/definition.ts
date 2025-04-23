import * as LSP from "vscode-languageserver-protocol";
import { DTLSContext } from "#lsp";
import { CssDocument } from "#css";

function getDefinitionFromCss(
  params: LSP.DefinitionParams,
  doc: CssDocument,
  context: DTLSContext,
) {
  const node = doc.getNodeAtPosition(params.position);
  const tokenName = node?.text;
  const token = context.tokens.get(tokenName);
  const spec = token && context.tokens.meta.get(token);

  if (tokenName && spec) {
    const uri = new URL(spec.path, params.textDocument.uri).href;
    const doc = context.documents.get(uri);
    if (doc.language === "json") {
      const range = doc.getRangeForTokenName(tokenName, spec.prefix);
      if (range) {
        return [{ uri, range }];
      }
    }
  }
  return [];
}

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
  const doc = context.documents.get(params.textDocument.uri);
  switch (doc.language) {
    case "json":
      throw new Error(
        "textDocument/definition not implemented for JSON documents",
      );
    case "css":
      return getDefinitionFromCss(params, doc, context);
  }
}
