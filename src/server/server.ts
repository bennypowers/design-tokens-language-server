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

import { writeAllSync } from 'jsr:@std/io/write-all';

export class Server {
  static messageCollector = "";

  static #traceLevel: TraceValues = TraceValues.Off;
  static #decoder = new TextDecoder();
  static #encoder = new TextEncoder();

  static #cancelled = new Set<RequestMessage['id']>;

  static async serve() {
    for await (const chunk of Deno.stdin.readable) {
      this.#handleChunk(chunk);
    }
  }

  static #handleChunk(chunk: Uint8Array<ArrayBuffer>) {
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

    this.messageCollector = this.messageCollector.slice(messageEnd);

    if (this.#cancelled.has(request.id))
      return Logger.debug`Skipping DEAD request ${request.id}`;

    if (request.id != null)
      Logger.debug`📥 (${request.id}): ${request.method ?? 'notification'}`;

    this.#handle(request);
  }

  static #requestQueue = new Set<RequestMessage>();

  static #processing = false;

  static async #handle(request: RequestMessage) {
    if (request.id && this.#cancelled.has(request.id)) return;
    this.#requestQueue.add(request);
    if (!this.#processing) {
      while (this.#requestQueue.size > 0) {
        this.#processing = true;
        const [currentRequest] = this.#requestQueue.values().take(1);
        if (currentRequest) {
          try {
            const result = await this.#result(currentRequest);
            this.#respond(currentRequest.id, result);
          } catch (error) {
            this.#respond(currentRequest.id, null, error as ResponseError);
          }
        }
      }
      this.#processing = false;
    }
  }

  static #result(request: RequestMessage): unknown | Promise<unknown> {
    this.#requestQueue.delete(request);
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
          this.#cancelled.add(id);
          const cancelledReq = this.#requestQueue.values().find(r => r.id === id);
          if (cancelledReq)
            this.#requestQueue.delete(cancelledReq);
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

  static #respond(id?: string|number|null, result?: unknown, error?: ResponseError) {
    if (!id && !result && !error) return;
    const pkg = { jsonrpc: "2.0", id, result, error };
    const message = JSON.stringify(pkg);
    const messageLength = this.#encoder.encode(message).byteLength;
    Logger.debug`Remaining Requests: ${[...this.#requestQueue]}`
    this.#print(`Content-Length: ${messageLength}\r\n\r\n${message}`);
  }

  /**
   * Deno.stdout.write is not guaranteed to write the whole buffer in a single call
   * using it led to unexpected bugs, bad responses, etc.
   * https://stackoverflow.com/a/79576657/2515275
   */
  static #print(input: string) {
    writeAllSync(
      Deno.stdout,
      new TextEncoder().encode(input),
    );
  }
}
