import {
  CompletionItem,
  CompletionParams,
  DidChangeTextDocumentParams,
  DidCloseTextDocumentParams,
  DidOpenTextDocumentParams,
  DocumentColorParams,
  HoverParams,
  InitializeParams,
  RequestMessage,
  ResponseError,
} from "vscode-languageserver-protocol";

import { Logger } from "./logger.ts";

import { initialize } from "./methods/initialize.ts";
import { didOpen } from "./methods/textDocument/didOpen.ts";
import { didChange } from "./methods/textDocument/didChange.ts";
import { didClose } from "./methods/textDocument/didClose.ts";

import { documentColor } from "./methods/textDocument/documentColor.ts";

import { hover } from "./methods/textDocument/hover.ts";

import { completion } from "./methods/textDocument/completion.ts";
import { resolve } from "./methods/completionItem/resolve.ts";

export class Server {
  static messageCollector = "";

  static #decoder = new TextDecoder();
  static #encoder = new TextEncoder();

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

    let result, error;

    try {
      result = await this.#result(request);
    } catch(err) {
      result = null
      error = err as ResponseError
    }

    this.#respond(request.id, result, error);

    this.messageCollector = this.messageCollector.slice(messageEnd);
  }

  static async #result(request: RequestMessage): Promise<unknown> {
    switch (request.method) {
      case "initialize": return initialize(request.params as InitializeParams);

      case "textDocument/didOpen": return didOpen(request.params as DidOpenTextDocumentParams);
      case "textDocument/didChange": return didChange(request.params as DidChangeTextDocumentParams);
      case "textDocument/didClose": return didClose(request.params as DidCloseTextDocumentParams);
      case "textDocument/documentColor": return documentColor(request.params as DocumentColorParams);

      case "textDocument/hover": return hover(request.params as HoverParams);
      case "textDocument/completion": return completion(request.params as CompletionParams);

      case "completionItem/resolve": return resolve(request.params as CompletionItem);
      default:
        return null;
    }
  }

  static #respond(id?: string|number|null, result?: unknown, error?: ResponseError) {
    if (id == null && !result && !error)
      return;
    const message = JSON.stringify({ jsonrpc: "2.0", id, result, error });
    const messageLength = this.#encoder.encode(message).byteLength;
    Logger.debug(message);
    const payload = `Content-Length: ${messageLength}\r\n\r\n${message}`;
    Deno.stdout.write(this.#encoder.encode(payload));
  }
}
