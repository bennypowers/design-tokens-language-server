import { RequestMessage, ResponseError } from "vscode-languageserver-protocol";

import { Logger } from "#logger";
import { Io } from "#server";

import { writeAllSync } from "@std/io/write-all";

/**
 * The Stdio class implements the Io interface for handling standard input and output.
 * It reads requests from stdin and writes responses to stdout.
 */
export class Stdio implements Io {
  #chunks = "";
  #decoder = new TextDecoder;
  #encoder = new TextEncoder;

  public async * requests() {
    for await (const chunk of Deno.stdin.readable) {
      try {
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

          yield JSON.parse(slice) as RequestMessage;
        }
      } catch(e) {
        Logger.error`STDIO READ ERROR: ${e}`
      }
    }
  }

  public respond(id?: RequestMessage['id'], result?: unknown, error?: ResponseError) {
    if (!id && !result && !error) return;
    const pkg = { jsonrpc: "2.0", id, result, error };
    const message = JSON.stringify(pkg);
    const messageLength = this.#encoder.encode(message).byteLength;
    const payload = `Content-Length: ${messageLength}\r\n\r\n${message}`;
    writeAllSync(Deno.stdout, this.#encoder.encode(payload));
  }
}
