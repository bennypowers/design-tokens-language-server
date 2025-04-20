import { Hover, HoverParams, MarkupContent, MarkupKind } from "vscode-languageserver-protocol";

import { tokens, getTokenMarkdown } from "#tokens";
import { documents, tsRangeToLspRange } from "#css";

export function hover(params: HoverParams): null | Hover {
  const node = documents.getNodeAtPosition(params.textDocument.uri, params.position);
  if (node) {
    const token = tokens.get(node.text);
    if (token) {
      const contents: MarkupContent = {
        value: getTokenMarkdown(node.text, token),
        kind: MarkupKind.Markdown,
      }
      return {
        contents,
        range: tsRangeToLspRange(node),
      };
    }
  }
  return null;
}
