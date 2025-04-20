import { RequestMessage, ResponseError } from "vscode-languageserver-protocol";

import { Logger } from "#logger";

import { writeAllSync } from "jsr:@std/io/write-all";

import { createQueue } from "@sv2dev/tasque";

import { Lsp } from "./lsp/lsp.ts";

export class Server {
  static #chunks = "";
  static #handlers: Lsp;
  static #queue = createQueue({ parallelize: 5 });
  static #decoder = new TextDecoder;
  static #encoder = new TextEncoder;

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
      await this.#handlers.process(request);
    else if (this.#handlers.isCancelledRequest(request.id))
      return this.#respond(request.id, null);
    else
      await this.#queue.add(async () => {
        try {
          if (!request.method.match(/^initialized?$/))
            await this.#handlers.isInitialized();
          const result = await this.#handlers.process(request);
          return this.#respond(request.id, result);
        } catch (error) {
          Logger.error`${error}`;
          this.#respond(request.id, null, error as ResponseError);
        }
      });
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
    this.#handlers = new Lsp();
    for await (const chunk of Deno.stdin.readable) {
      try {
        this.#handleChunk(chunk);
      } catch(e) {
        Logger.debug`CHUNK ERROR: ${e}`
      }
    }
  }
}
