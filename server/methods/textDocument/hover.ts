import { Hover, HoverParams, MarkupContent, MarkupKind } from "vscode-languageserver-protocol";

import { getCSSWordAtPosition } from "../../css/css.ts";
import { get } from "../../storage.ts";
import { getTokenMarkdown } from "../../markdown.ts";

export function hover(params: HoverParams): null | Hover {
  const { word, range } = getCSSWordAtPosition(
    params.textDocument.uri,
    params.position,
  );
  const token = get(word?.replace(/^--/, "") ?? "");
  if (token) {
    const contents: MarkupContent = {
      value: getTokenMarkdown(token),
      kind: MarkupKind.Markdown,
    }
    return { contents, range };
  }
  return null;
}
