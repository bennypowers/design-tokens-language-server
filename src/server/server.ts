import { Message, RequestMessage, ResponseError, SetTraceParams, TraceValues } from "vscode-languageserver-protocol";

import { Logger } from "./logger.ts";

import { initialize, SupportedMessage } from "./methods/initialize.ts";

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
  static #decoder = new TextDecoder;
  static #encoder = new TextEncoder;
  static #cancelled = new Set<RequestMessage['id']>;
  static #resolveInitialized: () => void;
  static #initialized = new Promise<void>(r => this.#resolveInitialized = r);

  static #cancelRequest(request: RequestMessage) {
    const { id } = request;
    this.#cancelled.add(id);
    Logger.debug(`📵 Cancel {id}`, { id });
    return null;
  }

  static #setTrace(params: SetTraceParams) {
    this.#traceLevel = params.value;
  }

  static #handleChunk(chunk: Uint8Array<ArrayBuffer>) {
    this.#chunks += this.#decoder.decode(chunk);

    while (this.#chunks.includes("\r\n\r\n")) {
      const [, lengthMatch] = this.#chunks.match(/Content-Length: (\d+)\r\n/) ?? [];
      if (lengthMatch == null) break;

      const contentLength = parseInt(lengthMatch);
      const messageStart = this.#chunks.indexOf("\r\n\r\n") + 4;
      const messageEnd = messageStart + contentLength;

      if (this.#chunks.length < messageEnd) break;

      const slice = this.#chunks.slice(messageStart, messageEnd);
      this.#chunks = this.#chunks.slice(messageEnd);

      try {
        const request = JSON.parse(slice) as RequestMessage;
        Logger.debug`📥 (${request.id ?? ''}): ${request.method ?? "notification"}`;
        this.#handle(request);
      } catch (error) {
        Logger.error`${error}`;
      }
    }
  }

  static async #handle(request: RequestMessage) {
    if (!request.id)
      await this.#process(request as SupportedMessage);
    else
      await this.#queue.add(async () => {
        try {
          if (this.#cancelled.has(request.id))
            return this.#respond(request.id, null);
          if (!request.method.match(/^initialized?$/))
            await this.#initialized;
          const result = await this.#process(request as SupportedMessage);
          return this.#respond(request.id, result);
        } catch (error) {
          Logger.error`${error}`;
          this.#respond(request.id, null, error as ResponseError);
        }
      });
  }

  static async #process(request: SupportedMessage): Promise<unknown> {
    if (Message.isRequest(request) && this.#cancelled.has(request.id)) return null;
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

  static #respond(id?: RequestMessage['id'], result?: unknown, error?: ResponseError) {
    if (!id && !result && !error) return;
    const pkg = { jsonrpc: "2.0", id, result, error };
    const message = JSON.stringify(pkg);
    const messageLength = this.#encoder.encode(message).byteLength;
    const payload = `Content-Length: ${messageLength}\r\n\r\n${message}`;
    writeAllSync(Deno.stdout, this.#encoder.encode(payload));
  }

  public static async serve() {
    for await (const chunk of Deno.stdin.readable) {
      try {
        this.#handleChunk(chunk);
      } catch(e) {
        Logger.debug`CHUNK ERROR: ${e}`
      }
    }
  }
}
