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
 * @returns The hover information containing the token's documentation and range.
 */
export function hover(params: HoverParams, context: DTLSContext): null | Hover {
  const node = context.documents.getNodeAtPosition(
    params.textDocument.uri,
    params.position,
  );
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
  return null;
}
