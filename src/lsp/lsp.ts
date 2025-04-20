import * as LSP from "vscode-languageserver-protocol";

import { documents } from "#css";
import { Logger } from "#logger";

import { initialize } from "./methods/initialize.ts";
import { documentColor } from "./methods/textDocument/documentColor.ts";
import { codeAction } from "./methods/textDocument/codeAction.ts";
import { diagnostic } from "./methods/textDocument/diagnostic.ts";
import { hover } from "./methods/textDocument/hover.ts";
import { completion } from "./methods/textDocument/completion.ts";
import { colorPresentation } from "./methods/textDocument/colorPresentation.ts";
import { resolve as completionItemResolve } from "./methods/completionItem/resolve.ts";
import { resolve as codeActionResolve } from "./methods/codeAction/resolve.ts";

const handlers = {
  ...documents.handlers,
  "initialize": initialize,
  "codeAction/resolve": codeActionResolve,
  "completionItem/resolve": completionItemResolve,
  "textDocument/codeAction": codeAction,
  "textDocument/colorPresentation": colorPresentation,
  "textDocument/completion": completion,
  "textDocument/diagnostic": diagnostic,
  "textDocument/documentColor": documentColor,
  "textDocument/hover": hover,
};

export type RequestId = LSP.RequestMessage['id'];

export type SupportedMethod = 'initialized'|'$/cancelRequest'|'$/setTrace'| keyof typeof handlers;

export type SupportedParams = Parameters<typeof handlers[keyof typeof handlers]>[0];

export type RequestTypeForMethod<M extends SupportedMethod> =
    M extends 'initialized' ? { method: M; params: LSP.InitializedParams }
  : M extends '$/cancelRequest' ? { method: M; params: Pick<LSP.RequestMessage, 'id'> }
  : M extends '$/setTrace' ? { method: M; params: LSP.SetTraceParams }
  : M extends keyof typeof handlers ? LSP.RequestMessage & { method: M; params: Parameters<typeof handlers[M]>[0]; }
  : never;

export type ResultFor<M extends SupportedMethod> = Awaited<
    M extends 'initialized' ? null
  : M extends '$/cancelRequest' ? null
  : M extends '$/setTrace' ? null
  : M extends keyof typeof handlers ? ReturnType<typeof handlers[M]>
  : never
>;

export type ResponseFor<M extends SupportedMethod> = Omit<
  LSP.ResponseMessage,
  'result'
> & { result: ResultFor<M> };

function isCancelRequest(
  request: Omit<LSP.NotificationMessage, 'jsonrpc'>,
): request is RequestTypeForMethod<'$/cancelRequest'> {
  return request.method === "$/cancelRequest";
}

function isSetTraceRequest(
  request: Omit<LSP.NotificationMessage, 'jsonrpc'>,
): request is RequestTypeForMethod<'$/setTrace'> {
  return request.method === "$/setTrace";
}

function isInitializedRequest(
  request: Omit<LSP.NotificationMessage, 'jsonrpc'>,
): request is RequestTypeForMethod<'initialized'> {
  return request.method === "initialized";
}

/**
 * The Lsp class is responsible for processing LSP requests and notifications.
 * It handles the initialization of the server, and the processing of various LSP methods.
 */
export class Lsp {
  #cancelled = new Set<RequestId>;
  #resolveInitialized!: () => void;
  #initialized = new Promise<void>(r => this.#resolveInitialized = r);
  #traceLevel: LSP.TraceValues = LSP.TraceValues.Off;
  #handlerMap = new Map(Object.entries(handlers));

  #cancelRequest(request: LSP.RequestMessage) {
    const { id } = request;
    this.#cancelled.add(id);
    Logger.debug`📵 Cancel ${id}`;
    return null;
  }

  #setTrace(params: LSP.SetTraceParams) {
    Logger.info`Set trace level to ${params.value}`;
    this.#traceLevel = params.value;
    return null;
  }

  /**
   * A promise which resolves when the server has completed initialization.
   */
  public initialized() {
    return this.#initialized;
  }

  /**
   * Informs the caller whether or not the request has been cancelled.
   */
  public isCancelledRequest(id: RequestId) {
    return this.#cancelled.has(id);
  }

  /**
   * Processes the given request and returns the result.
   *
   * @param request - The request to process.
   * @returns The result of the request.
   */
  public async process(request: LSP.RequestMessage): Promise<unknown> {
    if (LSP.Message.isRequest(request) && this.#cancelled.has(request.id))
      return null;
    else if (isInitializedRequest(request))
      return this.#resolveInitialized();
    else if (isSetTraceRequest(request))
    return this.#setTrace(request.params);
    else if (isCancelRequest(request))
      return this.#cancelRequest(request);
    else if (request.method)
      return await this.#handlerMap.get(request.method)?.(
        // deno-lint-ignore no-explicit-any
        request.params as any,
      ) ?? null;
  }
}
