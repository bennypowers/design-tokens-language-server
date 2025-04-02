import type { DidOpenTextDocumentParams, RequestMessage } from "vscode-languageserver-protocol";

import { documentTextCache } from "../../documents.ts";

export interface DidOpenRequestMessage extends RequestMessage {
  params: DidOpenTextDocumentParams;
}

export function didOpen(message: DidOpenRequestMessage): void {
  const { params } = message;
  documentTextCache.set(params.textDocument.uri, params.textDocument.text);
}
