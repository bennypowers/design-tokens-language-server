import type { RequestMessage, ResponseError } from "npm:vscode-languageserver-protocol";

import { Logger } from "./logger.ts";

import { initialize, type InitializeRequestMessage } from "./methods/initialize.ts";
import { completion, type CompletionRequestMessage } from "./methods/textDocument/completion.ts";
import { didChange, type DidChangeRequestMessage } from "./methods/textDocument/didChange.ts";
import { hover, type HoverRequestMessage } from "./methods/textDocument/hover.ts";
import { didOpen, DidOpenRequestMessage } from "./methods/textDocument/didOpen.ts";

export class Server {
  static messageCollector = "";

  static #decoder = new TextDecoder();

  static async serve() {
    Logger.write('Now serving');
    for await (const chunk of Deno.stdin.readable) {
      this.#handleChunk(chunk)
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

    const rawMessage = this.messageCollector.slice(
      messageStart,
      messageStart + contentLength,
    );
    const message = JSON.parse(rawMessage);

    Logger.write({ id: message.id, method: message.method });

    const result = await this.#handle(message);

    this.#respond(message, result);

    this.messageCollector = this.messageCollector.slice(messageStart + contentLength);
  }

  static #handle(request: RequestMessage): unknown | Promise<unknown> {
    switch (request.method) {
      case 'initialize':
        return initialize(request as InitializeRequestMessage);
      case 'textDocument/completion':
        return completion(request as CompletionRequestMessage);
      case 'textDocument/didOpen':
        return didOpen(request as DidOpenRequestMessage);
      case 'textDocument/didChange':
        return didChange(request as DidChangeRequestMessage);
      case 'textDocument/hover':
        return hover(request as HoverRequestMessage);
      default:
        return null;
    }
  }

  static #respond({ id, method }: RequestMessage, result?: unknown, error?: ResponseError) {
    result ??= null;
    if (!id && !result)
      return;
    const message = JSON.stringify({ id, result, error });
    const messageLength = new TextEncoder().encode(message).byteLength;
    const payload = `Content-Length: ${messageLength}\r\n\r\n${message}`;
    if (method === 'textDocument/completion')
      Logger.write(payload);
    Deno.stdout.write(new TextEncoder().encode(payload));
  }
}

