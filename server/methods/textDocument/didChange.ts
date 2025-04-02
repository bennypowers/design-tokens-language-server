import type { DidChangeTextDocumentParams, RequestMessage } from "vscode-languageserver-protocol";

import { documentTextCache } from "../../documents.ts";

export interface DidChangeRequestMessage extends RequestMessage {
  params: DidChangeTextDocumentParams;
}

export function didChange(message: DidChangeRequestMessage): void {
  const { params } = message;
  const [{ text }] = params.contentChanges;
  documentTextCache.set(params.textDocument.uri, text);
}
