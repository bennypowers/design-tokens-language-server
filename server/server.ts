import type {
  CompletionParams,
  DidChangeTextDocumentParams,
  DidOpenTextDocumentParams,
  DocumentColorParams,
  HoverParams,
  InitializeParams,
  RequestMessage,
  ResponseError,
} from "vscode-languageserver-protocol";

import { Logger } from "./logger.ts";

import { initialize } from "./methods/initialize.ts";
import { completion } from "./methods/textDocument/completion.ts";
import { didChange } from "./methods/textDocument/didChange.ts";
import { hover } from "./methods/textDocument/hover.ts";
import { didOpen } from "./methods/textDocument/didOpen.ts";
import { documentColor } from "./methods/textDocument/documentColor.ts";

export class Server {
  static messageCollector = "";

  static #decoder = new TextDecoder();
  static #encoder = new TextEncoder();

  static async serve() {
    Logger.write("Now serving");
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

    if (this.messageCollector.length < messageStart + contentLength) return;

    const slice = this.messageCollector.slice(messageStart, messageStart + contentLength);

    try {
      const message = JSON.parse(slice);

      Logger.write({ id: message.id, method: message.method });

      const result = await this.#handle(message);

      this.#respond(message, result);
    } catch(e) {
      Logger.write(`FAILED to write slice: ${slice}\n${e}`);
      if (e instanceof Error)
        Deno.stderr.write(this.#encoder.encode(e.toString()));
      else
        Deno.stderr.write(this.#encoder.encode(`FAILED to write slice: ${slice}`));
    }

    this.messageCollector = this.messageCollector.slice(
      messageStart + contentLength,
    );
  }

  static #handle(request: RequestMessage): unknown | Promise<unknown> {
    switch (request.method) {
      case "initialize":
        return initialize(request.params as InitializeParams);
      case "textDocument/documentColor":
        return documentColor(request.params as DocumentColorParams);
      case "textDocument/completion":
        return completion(request.params as CompletionParams);
      case "textDocument/didOpen":
        return didOpen(request.params as DidOpenTextDocumentParams);
      case "textDocument/didChange":
        return didChange(request.params as DidChangeTextDocumentParams);
      case "textDocument/hover":
        return hover(request.params as HoverParams);
      default:
        return null;
    }
  }

  static #respond({ id, method }: RequestMessage, result?: unknown, error?: ResponseError) {
    result ??= null;
    if (!id && !result) return;
    const message = JSON.stringify({ jsonrpc: '2.0', id, result, error });
    const messageLength = this.#encoder.encode(message).byteLength;
    const payload = `Content-Length: ${messageLength}\r\n\r\n${message}`;
    switch (method) {
      case "textDocument/completion":
        break;
      case "initialize":
        Logger.write(payload);
    }
    Deno.stdout.write(this.#encoder.encode(payload));
  }
}
