import {
CodeActionParams,
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
import { codeAction } from "./methods/textDocument/codeAction.ts";

const DEAD = Symbol('dead request');

export class Server {
  static messageCollector = "";

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

        case "textDocument/didOpen": return didOpen(request.params as DidOpenTextDocumentParams);
        case "textDocument/didChange": return didChange(request.params as DidChangeTextDocumentParams);
        case "textDocument/didClose": return didClose(request.params as DidCloseTextDocumentParams);
        case "textDocument/documentColor": return documentColor(request.params as DocumentColorParams);

        case "textDocument/hover": return hover(request.params as HoverParams);
        case "textDocument/completion": return completion(request.params as CompletionParams);
        case "textDocument/codeAction": return codeAction(request.params as CodeActionParams);

        case "completionItem/resolve": return resolve(request.params as CompletionItem);

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

  static async #respond(id?: string|number|null, result?: unknown, error?: ResponseError) {
    if (!((id == null && !result && !error) || (id != null && !this.#requests.has(id)))) {
      const pkg = { jsonrpc: "2.0", id, result, error };
      const message = JSON.stringify(pkg);
      const messageLength = this.#encoder.encode(message).byteLength;
      const request = this.#requests.get(id!);
      if (!request)
        Logger.debug(`↩️ {method}: {result}`, {
          method: (result as { method: string }).method,
          result
        });
      else if (request === DEAD)
        this.#requests.delete(id!);
      else if (id && error) {
        Logger.error(`↩️  ({id}): {method}\n{error}`, { id, error, method: request.method });
        this.#requests.delete(id);
      }
      else if (id) {
        Logger.debug(`↩️  ({id}): {method}\n{result}`, { id, method: request.method, result });
        this.#requests.delete(id);
      }
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
