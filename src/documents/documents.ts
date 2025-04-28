import * as LSP from "vscode-languageserver-protocol";

import { JsonDocument } from "#json";
import { CssDocument } from "#css";
import { DTLSContext } from "#lsp";
import { Logger } from "#logger";

class ENODOCError extends Error {
  constructor(public uri: LSP.DocumentUri) {
    super(`ENOENT: no Document found for ${uri}`);
  }
}

export class Documents {
  #map = new Map<LSP.DocumentUri, CssDocument | JsonDocument>();

  get handlers() {
    return {
      "textDocument/didOpen": (
        params: LSP.DidOpenTextDocumentParams,
        context: DTLSContext,
      ) => this.onDidOpen(params, context),
      "textDocument/didChange": (
        params: LSP.DidChangeTextDocumentParams,
        context: DTLSContext,
      ) => this.onDidChange(params, context),
      "textDocument/didClose": (
        params: LSP.DidCloseTextDocumentParams,
        context: DTLSContext,
      ) => this.onDidClose(params, context),
    } as const;
  }

  protected get allDocuments() {
    return [...this.#map.values()];
  }

  add(doc: CssDocument | JsonDocument) {
    this.#map.set(doc.uri, doc);
  }

  onDidOpen(params: LSP.DidOpenTextDocumentParams, context: DTLSContext) {
    const { uri, languageId, version, text } = params.textDocument;
    switch (languageId) {
      case "json":
        this.add(JsonDocument.create(context, uri, text, version));
        break;
      case "css":
        this.add(CssDocument.create(context, uri, text, version));
        break;
      default:
        throw new Error(
          `Unsupported language: ${params.textDocument.languageId}`,
        );
    }
    Logger.debug`📖 Opened ${uri}`;
  }

  onDidChange(params: LSP.DidChangeTextDocumentParams, _context: DTLSContext) {
    const { uri, version } = params.textDocument;
    const doc = this.get(uri);
    doc.update(params.contentChanges, version);
  }

  onDidClose(params: LSP.DidCloseTextDocumentParams, _: DTLSContext) {
    this.#map.delete(params.textDocument.uri);
  }

  get(uri: LSP.DocumentUri) {
    const doc = this.#map.get(uri);
    if (!doc) {
      throw new ENODOCError(uri);
    }
    return doc;
  }

  getAll(languageId: "json"): JsonDocument[];
  getAll(languageId: "css"): JsonDocument[];
  getAll(): (CssDocument | JsonDocument)[];
  getAll(languageId?: "json" | "css"): (CssDocument | JsonDocument)[] {
    if (languageId) {
      return this.allDocuments.filter((doc) => doc.language === languageId);
    }
    return this.allDocuments;
  }
}
