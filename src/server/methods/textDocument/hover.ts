import { Hover, HoverParams, MarkupContent, MarkupKind } from "vscode-languageserver-protocol";

import { get } from "../../storage.ts";
import { getTokenMarkdown } from "../../markdown.ts";
import { documents, tsRangeToLspRange } from "../../css/documents.ts";

export function hover(params: HoverParams): null | Hover {
  const node = documents.getNodeAtPosition(params.textDocument.uri, params.position);
  if (node) {
    const token = get(node.text);
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
