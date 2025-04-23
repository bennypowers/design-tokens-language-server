import {
  Hover,
  HoverParams,
  MarkupContent,
  MarkupKind,
} from "vscode-languageserver-protocol";

import { getTokenMarkdown } from "#tokens";
import { tsRangeToLspRange } from "#css";
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
  if (doc.language === "css") {
    const node = doc.getNodeAtPosition(params.position);
    if (node) {
      if (context.tokens.has(node.text)) {
        const contents: MarkupContent = {
          value: getTokenMarkdown(node.text, context.tokens.get(node.text)!),
          kind: MarkupKind.Markdown,
        };
        return {
          contents,
          range: tsRangeToLspRange(node),
        };
      }
    }
  }
  return null;
}
