import { Hover, HoverParams, MarkupContent, MarkupKind } from "vscode-languageserver-protocol";

import { getCssSyntaxNodeAtPosition, tsNodeToRange } from "../../css/css.ts";
import { get } from "../../storage.ts";
import { getTokenMarkdown } from "../../markdown.ts";

export function hover(params: HoverParams): null | Hover {
  const node = getCssSyntaxNodeAtPosition(params.textDocument.uri, params.position);
  if (node) {
    const token = get(node.text.replace(/^--/, "") ?? "");
    if (token) {
      const contents: MarkupContent = {
        value: getTokenMarkdown(token),
        kind: MarkupKind.Markdown,
      }
      return {
        contents,
        range: tsNodeToRange(node),
      };
    }
  }
  return null;
}
