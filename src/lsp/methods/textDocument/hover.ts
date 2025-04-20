import {
  Hover,
  HoverParams,
  MarkupContent,
  MarkupKind,
} from "vscode-languageserver-protocol";

import { getTokenMarkdown, tokens } from "#tokens";
import { documents, tsRangeToLspRange } from "#css";

/**
 * Generates hover information for design tokens.
 *
 * @param params - The parameters for the hover request.
 * @returns The hover information containing the token's documentation and range.
 */
export function hover(params: HoverParams): null | Hover {
  const node = documents.getNodeAtPosition(
    params.textDocument.uri,
    params.position,
  );
  if (node) {
    if (tokens.has(node.text)) {
      const contents: MarkupContent = {
        value: getTokenMarkdown(node.text, tokens.get(node.text)!),
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
