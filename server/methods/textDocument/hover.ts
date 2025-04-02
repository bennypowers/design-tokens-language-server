import type { Hover, HoverParams } from "vscode-languageserver-protocol";

import { getCSSWordAtPosition } from "../../css/css.ts";
import { get } from "../../storage.ts";
import { getTokenMarkdown } from "../../token.ts";
import { Logger } from "../../logger.ts";

export function hover(params: HoverParams): null | Hover {
  const { word, range } = getCSSWordAtPosition(params.textDocument.uri, params.position);
  const token = get(word?.replace(/^--/, '') ?? '');
  if (token) {
    Logger.write(`Hover word: ${word}`);
    return {
      contents: getTokenMarkdown(token),
      range,
    }
  }
  return null;
}
