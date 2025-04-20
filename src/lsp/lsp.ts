import * as LSP from "vscode-languageserver-protocol";

import { initialize, SupportedMessage } from "./methods/initialize.ts";

import { documentColor } from "./methods/textDocument/documentColor.ts";
import { codeAction } from "./methods/textDocument/codeAction.ts";
import { diagnostic } from "./methods/textDocument/diagnostic.ts";
import { hover } from "./methods/textDocument/hover.ts";
import { completion } from "./methods/textDocument/completion.ts";
import { resolve as completionItemResolve } from "./methods/completionItem/resolve.ts";
import { resolve as codeActionResolve } from "./methods/codeAction/resolve.ts";

import { documents } from "#css";
import { Logger } from "#logger";

type RequestId = LSP.RequestMessage['id'];

export class Lsp {
  #cancelled = new Set<RequestId>;
  #resolveInitialized!: () => void;

  #initialized = new Promise<void>(r => this.#resolveInitialized = r);

  #traceLevel: LSP.TraceValues = LSP.TraceValues.Off;

  #cancelRequest(request: LSP.RequestMessage) {
    const { id } = request;
    this.#cancelled.add(id);
    Logger.debug(`📵 Cancel {id}`, { id });
    return null;
  }

  #setTrace(params: LSP.SetTraceParams) {
    this.#traceLevel = params.value;
  }

  public isInitialized() {
    return this.#initialized;
  }

  public isCancelledRequest(id: RequestId) {
    return this.#cancelled.has(id);
  }

  public async process(_request: LSP.RequestMessage): Promise<unknown> {
    const request = _request as SupportedMessage;
    if (LSP.Message.isRequest(request) && this.#cancelled.has(request.id)) return null;
    switch (request.method) {
      case "initialize": return await initialize(request.params);
      case "initialized": return this.#resolveInitialized();

      case "textDocument/didOpen": return documents.onDidOpen(request.params);
      case "textDocument/didChange": return documents.onDidChange(request.params);
      case "textDocument/didClose": return documents.onDidClose(request.params);
      case "textDocument/diagnostic": return diagnostic(request.params);
      case "textDocument/documentColor": return documentColor(request.params);

      case "textDocument/hover": return hover(request.params);
      case "textDocument/completion": return completion(request.params);
      case "textDocument/codeAction": return codeAction(request.params);

      case "completionItem/resolve": return completionItemResolve(request.params);
      case "codeAction/resolve": return codeActionResolve(request.params);

      case "$/setTrace": return this.#setTrace(request.params);
      case "$/cancelRequest": return this.#cancelRequest(request.params)

      default:
        return null;
    }
  }

}
