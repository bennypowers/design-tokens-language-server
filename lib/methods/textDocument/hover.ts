import { Hover, HoverParams, RequestMessage } from "vscode-languageserver-protocol";
import { Logger } from "../../logger.ts";
import { getCSSWordAtPosition } from "../../css/css.ts";
import { get } from "../../storage.ts";
import { getTokenMarkdown } from "../../token.ts";

export interface HoverRequestMessage extends RequestMessage {
  params: HoverParams;
}

export function hover(message: HoverRequestMessage): null | Hover {
  const word = getCSSWordAtPosition(
    message.params.textDocument.uri,
    message.params.position,
  )?.replace(/^--/, '');
  const token = get(word ?? '');
  if (token) {
    Logger.write(token)
    return {
      contents: getTokenMarkdown(token),
    }
  }
  return null;
}
