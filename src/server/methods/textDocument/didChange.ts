import type { DidChangeTextDocumentParams } from "vscode-languageserver-protocol";

import { documentTextCache } from "../../documents.ts";

export function didChange(params: DidChangeTextDocumentParams): void {
  const [{ text }] = params.contentChanges;
  documentTextCache.set(params.textDocument.uri, text);
}
