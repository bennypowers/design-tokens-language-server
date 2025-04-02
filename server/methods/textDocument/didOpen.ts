import type { DidOpenTextDocumentParams } from "vscode-languageserver-protocol";

import { documentTextCache } from "../../documents.ts";

export function didOpen(params: DidOpenTextDocumentParams): void {
  documentTextCache.set(params.textDocument.uri, params.textDocument.text);
}
