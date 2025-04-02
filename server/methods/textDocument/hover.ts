import type { Hover, HoverParams } from "vscode-languageserver-protocol";

import { getCSSWordAtPosition } from "../../css/css.ts";
import { get } from "../../storage.ts";
import { getTokenMarkdown } from "../../token.ts";

export function hover(params: HoverParams): null | Hover {
  const { word, range } = getCSSWordAtPosition(
    params.textDocument.uri,
    params.position,
  );
  const token = get(word?.replace(/^--/, "") ?? "");
  if (token) {
    return {
      contents: {
        value: getTokenMarkdown(token),
        kind: "markdown",
      },
      range,
    };
  }
  return null;
}
