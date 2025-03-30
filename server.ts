import type { RequestMessage, ResponseMessage } from "./main.ts";

import { Logger } from "./logger.ts";

import { Buffer } from "node:buffer";

type RequestMethodName = "initialize";

type ServerCapabilities = Record<string, unknown>;

interface InitializeResult {
  capabilities: ServerCapabilities;
  serverInfo?: {
    name: string;
    version?: string;
  };
}

interface Response {
  id?: number;
  result: Record<string, unknown>;
}

type RequestMethod = (message: RequestMessage) => Record<string, unknown>;
type NotificationMethod = (message: RequestMessage) => Record<string, unknown>;

export class Server {
  static buffer = "";

  static #log = Logger;

  static #RequestMap: Record<string, RequestMethod | NotificationMethod> = {
    initialize(_message: RequestMessage) {
      return {
        capabilities: {},
        serverInfo: {
          name: "design-tokens-languageserver",
          version: "0.0.1",
        },
      };
    },
  };

  static async serve() {
    for await (const chunk of Deno.stdin.readable) {
      this.buffer += chunk;
      while (true) {
        const [, lengthMatch] =
          this.buffer.match(/Content-Length: (\d+)\r\n/) ?? [];
        if (lengthMatch == null) break;
        const contentLength = parseInt(lengthMatch);
        const messageStart = this.buffer.indexOf("\r\n\r\n") + 4;
        if (this.buffer.length < messageStart + contentLength) break;
        const rawMessage = this.buffer.slice(
          messageStart,
          messageStart + contentLength,
        );
        const message = JSON.parse(rawMessage);

        this.#log.write({ id: message.id, method: message.method });

        const method = this.#RequestMap[message.method];
        if (method) {
          const result = method(message);
          if (result != null) {
            this.#respond(message.id, result);
          }
        }

        this.buffer.slice(messageStart + contentLength);
      }
    }
  }

  static #respond(
    id: ResponseMessage["id"],
    result: Record<string, unknown> | null,
  ) {
    const message = JSON.stringify({ id, result });
    const messageLength = Buffer.byteLength(message);
    const payload = `Content-Length: ${messageLength}\r\n\r\n${message}`;
    this.#log.write(payload);
    Deno.stdout.write(new TextEncoder().encode(payload));
  }
}
