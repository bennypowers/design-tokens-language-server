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

import { writeAllSync } from "jsr:@std/io/write-all";

import { createQueue } from "@sv2dev/tasque";

export class Server {
  static #chunks = "";
  static #queue = createQueue({ parallelize: 5 });
  static #traceLevel: TraceValues = TraceValues.Off;
  static #decoder = new TextDecoder();
  static #encoder = new TextEncoder();
  static #requestControllers = new Map<RequestMessage['id'], AbortController>();

  static async serve() {
    for await (const chunk of Deno.stdin.readable) {
      this.#handleChunk(chunk);
    }
  }

  static #handleChunk(chunk: Uint8Array<ArrayBuffer>) {
    this.#chunks += this.#decoder.decode(chunk);
    const [, lengthMatch] = this.#chunks.match(/Content-Length: (\d+)\r\n/) ??
      [];

    if (lengthMatch == null) return;

    const contentLength = parseInt(lengthMatch);
    const messageStart = this.#chunks.indexOf("\r\n\r\n") + 4;
    const messageEnd = messageStart + contentLength;

    if (this.#chunks.length < messageStart + contentLength) return;

    const slice = this.#chunks.slice(messageStart, messageEnd);
    const request = JSON.parse(slice) as RequestMessage;
    this.#chunks = this.#chunks.slice(messageEnd);

    if (request.id != null) {
      Logger.debug`📥 (${request.id}): ${request.method ?? "notification"}`;
    }

    this.#handle(request);
  }

  static async #handle(request: RequestMessage) {
    if (request.id && this.#requestControllers.get(request.id)?.signal.aborted) return;
    const { id } = request;
    const ctrl = new AbortController();
    this.#requestControllers.set(id, ctrl);
    this.#queue.add(async () => {
      try {
        const result = await this.#result(request);
        return this.#respond(id, result);
      } catch (error) {
        this.#respond(id, null, error as ResponseError);
      }
    }, ctrl.signal);
  }

  static async #result(request: RequestMessage): Promise<unknown> {
    try {
      switch (request.method) {
        case "initialize":
          return initialize(request.params as InitializeParams);

        case "textDocument/didOpen":
          return documents.onDidOpen(
            request.params as DidOpenTextDocumentParams,
          );
        case "textDocument/didChange":
          return documents.onDidChange(
            request.params as DidChangeTextDocumentParams,
          );
        case "textDocument/didClose":
          return documents.onDidClose(
            request.params as DidCloseTextDocumentParams,
          );
        case "textDocument/diagnostic":
          return diagnostic(request.params as DocumentDiagnosticParams);
        case "textDocument/documentColor":
          return documentColor(request.params as DocumentColorParams);

        case "textDocument/hover":
          return hover(request.params as HoverParams);
        case "textDocument/completion":
          return completion(request.params as CompletionParams);
        case "textDocument/codeAction":
          return codeAction(request.params as CodeActionParams);

        case "completionItem/resolve":
          return completionItemResolve(request.params as CompletionItem);
        case "codeAction/resolve":
          return codeActionResolve(request.params as CodeAction);

        case "$/setTrace":
          return (params: SetTraceParams) => this.#traceLevel = params.value;

        case "$/cancelRequest": {
          const { id } = request.params as RequestMessage;
          Logger.debug(`📵 Cancel {id}`, { id });
          this.#requestControllers.get(id)?.abort(request.method);
          return null;
        }

        default:
          return null;
      }
    } catch (e) {
      Logger.error`${e}`;
      return null;
    }
  }

  static #respond(
    id?: string | number | null,
    result?: unknown,
    error?: ResponseError,
  ) {
    if (!id && !result && !error) return;
    const pkg = { jsonrpc: "2.0", id, result, error };
    const message = JSON.stringify(pkg);
    const messageLength = this.#encoder.encode(message).byteLength;
    writeAllSync(
      Deno.stdout,
      this.#encoder.encode(
        `Content-Length: ${messageLength}\r\n\r\n${message}`,
      ),
    );
  }
}
