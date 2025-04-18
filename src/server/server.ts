import { RequestMessage, ResponseError, SetTraceParams, TraceValues, Message, ResponseMessage } from "vscode-languageserver-protocol";

import { Logger } from "./logger.ts";

import { initialize, SupportedRequestMessage } from "./methods/initialize.ts";

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
  static #requests = new Set<RequestMessage['id']>;

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

    if (request.id != null)
      Logger.debug`📥 (${request.id}): ${request.method ?? "notification"}`;

    this.#handle(request);
  }

  static async #handle(request: RequestMessage) {
    if (request.id && this.#cancelled.has(request.id)) return;
    const { id } = request;
    this.#requests.add(id)
    await this.#queue.add(async () => {
      try {
        const result = await this.#process(request as SupportedRequestMessage);
        if (!Message.isNotification(request))
          return this.#respond(id, result);
      } catch (error) {
        Logger.error`${error}`;
        this.#respond(id, null, error as ResponseError);
      } finally {
        this.#requests.delete(id);
      }
    });
  }

  static async #process(request: SupportedRequestMessage): Promise<unknown> {
    switch (request.method) {
      case "initialize": return await initialize(request.params);

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

  static #respond(id?: ResponseMessage['id'], result?: unknown, error?: ResponseError) {
    // if (!id && !result && !error) return;
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
