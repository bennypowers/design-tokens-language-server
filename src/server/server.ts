import {
  CodeAction,
  CodeActionParams,
  CompletionItem,
  CompletionParams,
  DidChangeTextDocumentParams,
  DidCloseTextDocumentParams,
  DidOpenTextDocumentParams,
  DocumentColorParams,
  DocumentDiagnosticParams,
  HoverParams,
  InitializeParams,
  RequestMessage,
  ResponseError,
  SetTraceParams,
  TraceValues,
} from "vscode-languageserver-protocol";

import { Logger } from "./logger.ts";

import { initialize } from "./methods/initialize.ts";

import { documentColor } from "./methods/textDocument/documentColor.ts";
import { codeAction } from "./methods/textDocument/codeAction.ts";
import { diagnostic } from "./methods/textDocument/diagnostic.ts";
import { hover } from "./methods/textDocument/hover.ts";
import { completion } from "./methods/textDocument/completion.ts";
import { resolve as completionItemResolve } from "./methods/completionItem/resolve.ts";
import { resolve as codeActionResolve } from "./methods/codeAction/resolve.ts";
import { documents } from "./css/documents.ts";

const DEAD = Symbol('dead request');

export class Server {
  static messageCollector = "";

  static #traceLevel: TraceValues = TraceValues.Off;
  static #decoder = new TextDecoder();
  static #encoder = new TextEncoder();

  static #requests = new Map<RequestMessage['id'], RequestMessage | typeof DEAD>;

  static async serve() {
    for await (const chunk of Deno.stdin.readable) {
      this.#handleChunk(chunk);
    }
  }

  static async #handleChunk(chunk: Uint8Array<ArrayBuffer>) {
    this.messageCollector += this.#decoder.decode(chunk);
    const [, lengthMatch] =
      this.messageCollector.match(/Content-Length: (\d+)\r\n/) ?? [];

    if (lengthMatch == null) return;

    const contentLength = parseInt(lengthMatch);

    const messageStart = this.messageCollector.indexOf("\r\n\r\n") + 4;
    const messageEnd = messageStart + contentLength;

    if (this.messageCollector.length < messageStart + contentLength) return;

    const slice = this.messageCollector.slice(messageStart, messageEnd);

    const request = JSON.parse(slice) as RequestMessage;

    if (!(request.id !== null && this.#requests.get(request.id) === DEAD)) {
      Logger.debug(`📥 ({id}): {method}`, { id: request.id ?? 'notification', method: request.method });
      this.#respond(request.id, ...await this.#handle(request));
    }

    this.messageCollector = this.messageCollector.slice(messageEnd);
  }

  static async #handle(request: RequestMessage) {
    this.#requests.set(request.id, request);

    let result, error;

    try {
      result = await this.#result(request);
    } catch(err) {
      result = null
      error = err as ResponseError
    }

    return [result, error];

  }

  static #result(request: RequestMessage): unknown | Promise<unknown> {
    try {
      switch (request.method) {
        case "initialize": return initialize(request.params as InitializeParams);

        case "textDocument/didOpen": return documents.onDidOpen(request.params as DidOpenTextDocumentParams);
        case "textDocument/didChange": return documents.onDidChange(request.params as DidChangeTextDocumentParams);
        case "textDocument/didClose": return documents.onDidClose(request.params as DidCloseTextDocumentParams);
        case "textDocument/diagnostic": return diagnostic(request.params as DocumentDiagnosticParams);
        case "textDocument/documentColor": return documentColor(request.params as DocumentColorParams);

        case "textDocument/hover": return hover(request.params as HoverParams);
        case "textDocument/completion": return completion(request.params as CompletionParams);
        case "textDocument/codeAction": return codeAction(request.params as CodeActionParams);

        case "completionItem/resolve": return completionItemResolve(request.params as CompletionItem);
        case "codeAction/resolve": return codeActionResolve(request.params as CodeAction);

        case "$/setTrace": return (params: SetTraceParams) => this.#traceLevel = params.value;

        case "$/cancelRequest": {
          const { id } = (request.params as RequestMessage)
          Logger.debug(`📵 Cancel {id}`, { id });
          this.#requests.delete(id);
          return null;
        }

        default:
          return null;
      }
    } catch(e) {
      Logger.error(`{e}`, {e});
      return null;
    }
  }

  static #logTrace(message: string, verbose: unknown) {
    Logger.debug`${message}:\n${verbose}`;
    if (this.#traceLevel !== TraceValues.Off) {
      if (this.#traceLevel !== TraceValues.Verbose)
        verbose = undefined;
      const pkg = { jsonrpc: "2.0", method: '$/logTrace', params: { message, verbose } };
      const payload = JSON.stringify(pkg);
      const messageLength = this.#encoder.encode(payload).byteLength;
      this.#print(`Content-Length: ${messageLength}\r\n\r\n${payload}`);
    }
  }

  static async #respond(id?: string|number|null, result?: unknown, error?: ResponseError) {
    if (!((id == null && !result && !error) || (id != null && !this.#requests.has(id)))) {
      const pkg = { jsonrpc: "2.0", id, result, error };
      const message = JSON.stringify(pkg);
      const messageLength = this.#encoder.encode(message).byteLength;
      const request = this.#requests.get(id!);
      if (request === DEAD)
        return;
      if (!request)
        this.#logTrace(`↩️ ${(result as { method: string }).method}`, result);
      else if (error)
        Logger.error`↩️  (${id}): ${request.method}\n${error}`;
      else if (id)
        this.#logTrace(`↩️  (${id}): ${request.method}`, result);

      if (id)
        this.#requests.delete(id!);

      await this.#print(`Content-Length: ${messageLength}\r\n\r\n${message}`);
    }
  }

  static #lastWrite: undefined | Promise<unknown>;

  /**
   * Deno.stdout.write is not guaranteed to write the whole buffer in a single call
   * using it led to unexpected bugs, bad responses, etc.
   * https://stackoverflow.com/a/79576657/2515275
   */
  static async #print(input: string) {
    if (this.#lastWrite) await this.#lastWrite;
    const stream = new Blob([input]).stream();
    this.#lastWrite = stream.pipeTo(Deno.stdout.writable, { preventClose: true })
  }
}
