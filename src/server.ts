import { RequestMessage, ResponseError } from "vscode-languageserver-protocol";

import { Logger } from "#logger";

import { createQueue } from "@sv2dev/tasque";

import { Lsp } from "#lsp";
import { Stdio } from "./server/stdio.ts";

export interface Io {
  /**
   * Requests are received from the client.
   * This is an async iterable that yields request messages.
   * The implementation of this method is responsible for reading from the appropriate input stream (e.g., stdin).
   */
  requests(): AsyncIterable<RequestMessage>;
  /**
   * Responds to the client with the result of the request.
   * The implementation of this method is responsible for writing to the appropriate output stream (e.g., stdout).
   */
  respond(id?: RequestMessage['id'], result?: unknown, error?: ResponseError): void | Promise<void>
}

export interface StdioOptions {
  io: 'stdio';
}

/**
 * The server class is responsible for handling the communication between the LSP and the client.
 *
 * It uses the Io interface to handle the communication, and the Lsp class to process the requests.
 */
export class Server {
  static #lsp: Lsp;
  static #io: Io;
  static #queue = createQueue({ parallelize: 5 });

  /**
   * The serve method is the entry point for the server.
   * It initializes the Lsp and Io instances, and starts listening for requests.
   */
  public static async serve(options: StdioOptions) {
    this.#lsp = new Lsp();

    switch (options.io) {
      case 'stdio':
        this.#io = new Stdio();
        break;
      default:
        throw new Error(`Unsupported IO type: ${options.io}`);
    }

    for await (const request of this.#io.requests()) {
      Logger.debug`${request.id ? `📥 (${request.id})` : `🔔`}: ${request.method ?? "notification"}`;
      if (!request.id)
        await this.#lsp.process(request);
      else if (this.#lsp.isCancelledRequest(request.id))
        return this.#io.respond(request.id, null);
      else
        await this.#queue.add(async () => {
          try {
            if (!request.method.match(/^initialized?$/))
              await this.#lsp.initialized();
            const result = await this.#lsp.process(request);
            if (request.id)
              Logger.debug`🚢 (${request.id}): ${request.method}`;
            return this.#io.respond(request.id, result);
          } catch (error) {
            Logger.error`${error}`;
            this.#io.respond(request.id, null, error as ResponseError);
          }
        });
      }
  }
}
