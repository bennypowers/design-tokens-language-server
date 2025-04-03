import type { DidCloseTextDocumentParams } from "vscode-languageserver-protocol";

import { documentTextCache } from "../../documents.ts";

export function didClose(params: DidCloseTextDocumentParams): void {
  documentTextCache.delete(params.textDocument.uri);
}
