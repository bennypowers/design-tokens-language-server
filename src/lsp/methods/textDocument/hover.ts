import {
  Hover,
  HoverParams,
  MarkupContent,
  MarkupKind,
} from "vscode-languageserver-protocol";

import { getTokenMarkdown } from "#tokens";
import { DTLSContext } from "#lsp";

/**
 * Generates hover information for design tokens.
 *
 * @param params - The parameters for the hover request.
 * @param context - The context containing design tokens and other information.
 * @returns The hover information containing the token's documentation and range.
 */
export function hover(params: HoverParams, context: DTLSContext): null | Hover {
  const doc = context.documents.get(params.textDocument.uri);
  const result = doc.getHoverTokenAtPosition(params.position);
  if (result) {
    const { name, token, range } = result;
    const contents: MarkupContent = {
      value: getTokenMarkdown(name, token),
      kind: MarkupKind.Markdown,
    };
    return { contents, range };
  } else {
    return null;
  }
}
