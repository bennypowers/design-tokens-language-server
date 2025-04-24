import {
  NotificationMessage,
  RequestMessage,
  ResponseError,
} from "vscode-languageserver-protocol";

import { Logger } from "#logger";

import { createQueue } from "@sv2dev/tasque";

import { Lsp } from "#lsp";
import { Documents } from "#documents";
import { Tokens } from "#tokens";
import { Workspaces } from "#workspaces";

import { Stdio } from "./stdio.ts";

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
  respond(
    id?: RequestMessage["id"],
    result?: unknown,
    error?: ResponseError,
  ): void | Promise<void>;

  /**
   * Sends a server notification to the client.
   */
  notify(message: NotificationMessage): void | Promise<void>;

  /**
   * Pushes a request message to the client.
   */
  push(message: Omit<RequestMessage, "jsonrpc" | "id">): void | Promise<void>;
}

export interface StdioOptions {
  io: "stdio";
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

  public static notify(message: NotificationMessage) {
    this.#io.notify(message);
  }

  public static push(message: Omit<RequestMessage, "jsonrpc" | "id">) {
    this.#io.push(message);
  }

  /**
   * The serve method is the entry point for the server.
   * It initializes the Lsp and Io instances, and starts listening for requests.
   */
  public static async serve(options: StdioOptions) {
    const documents = new Documents();
    const tokens = new Tokens();
    const workspaces = new Workspaces();
    this.#lsp = new Lsp(documents, workspaces, tokens);

    switch (options.io) {
      case "stdio":
        this.#io = new Stdio();
        break;
      default:
        throw new Error(`Unsupported IO type: ${options.io}`);
    }

    for await (const request of this.#io.requests()) {
      Logger.debug`${
        request.id != null ? `📩 (${request.id})` : `🔔`
      }: ${request.method}`;
      if (request.id == null) {
        await this.#lsp.process(request);
      } else if (this.#lsp.isCancelledRequest(request.id)) {
        return this.#io.respond(request.id, null);
      } else {
        await this.#queue.add(async () => {
          try {
            if (!request.method.match(/^initialized?$/)) {
              await this.#lsp.initialized();
            }
            const result = await this.#lsp.process(request);
            if (request.id != null) {
              Logger.debug`🚢 (${request.id}): ${request.method}`;
            } else {
              Logger.debug`🚀 (${request.method}) ${result}`;
            }
            return this.#io.respond(request.id, result);
          } catch (error) {
            Logger.error`${error}`;
            this.#io.respond(request.id, null, error as ResponseError);
          }
        });
      }
    }
  }
}
