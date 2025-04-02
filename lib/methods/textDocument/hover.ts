import type { Hover, HoverParams, RequestMessage } from "vscode-languageserver-protocol";

import { getCSSWordAtPosition } from "../../css/css.ts";
import { get } from "../../storage.ts";
import { getTokenMarkdown } from "../../token.ts";
import { Logger } from "../../logger.ts";

export interface HoverRequestMessage extends RequestMessage {
  params: HoverParams;
}

export function hover(message: HoverRequestMessage): null | Hover {
  const { word, range } =
    getCSSWordAtPosition(message.params.textDocument.uri, message.params.position);
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
