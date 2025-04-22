import {
  Message,
  NotificationMessage,
  RequestMessage,
  ResponseError,
} from "vscode-languageserver-protocol";

import { Logger } from "#logger";
import { Io } from "#server";

import { writeAllSync } from "@std/io/write-all";

/**
 * The Stdio class implements the Io interface for handling standard input and output.
 * It reads requests from stdin and writes responses to stdout.
 */
export class Stdio implements Io {
  #chunks = "";
  #decoder = new TextDecoder();
  #encoder = new TextEncoder();
  #lastId = 0;

  public async *requests() {
    for await (const chunk of Deno.stdin.readable) {
      try {
        this.#chunks += this.#decoder.decode(chunk);

        while (this.#chunks.includes("\r\n\r\n")) {
          const [, lengthMatch] =
            this.#chunks.match(/Content-Length: (\d+)\r\n/) ?? [];
          if (lengthMatch == null) break;

          const contentLength = parseInt(lengthMatch);
          const messageStart = this.#chunks.indexOf("\r\n\r\n") + 4;
          const messageEnd = messageStart + contentLength;

          if (this.#chunks.length < messageEnd) break;

          const slice = this.#chunks.slice(messageStart, messageEnd);
          this.#chunks = this.#chunks.slice(messageEnd);

          yield JSON.parse(slice) as RequestMessage;
        }
      } catch (e) {
        Logger.error`STDIO READ ERROR: ${e}`;
      }
    }
  }

  public notify(message: Omit<NotificationMessage, "jsonrpc">) {
    this.#sendJsonRpcMessage({ jsonrpc: "2.0", ...message });
  }

  public push(message: Omit<RequestMessage, "jsonrpc" | "id">) {
    const id = this.#lastId + 1;
    this.#sendJsonRpcMessage({
      jsonrpc: "2.0",
      id,
      ...message,
    } as RequestMessage);
  }

  public respond(
    id?: RequestMessage["id"],
    result?: unknown,
    error?: ResponseError,
  ) {
    if (!id && !result && !error) return;
    return this.#sendJsonRpcMessage({
      jsonrpc: "2.0",
      ...(id !== undefined) && { id },
      ...(result !== undefined) && { result },
      ...error && { error },
    });
  }

  #sendJsonRpcMessage(message: Message) {
    if (Message.isResponse(message) || Message.isRequest(message)) {
      const { id } = message;
      if (id !== null) {
        this.#lastId = typeof id === "number" ? id : parseInt(id);
      }
    }
    const messageString = JSON.stringify(message);
    const messageLength = this.#encoder.encode(messageString).byteLength;
    const payload = `Content-Length: ${messageLength}\r\n\r\n${messageString}`;
    writeAllSync(Deno.stdout, this.#encoder.encode(payload));
  }
}
