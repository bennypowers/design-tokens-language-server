import { Token } from "style-dictionary";
import { Documents } from "#css";
import { TokenMap } from "#tokens";
import { Position, Range } from "vscode-languageserver-protocol";

import { RequestTypeForMethod, ResponseFor, SupportedMethod } from "#lsp";

import testTokens from "../test/tokens.json" with { type: "json" };

/**
 * Test Documents for managing text documents.
 *
 * This class extends the Documents class and provides methods to create
 * mock text documents.
 * It also provides methods to get the text and ranges of specific strings
 * within the documents.
 */
class TestDocuments extends Documents {
  #tokens: TokenMap;

  constructor(tokens: TokenMap) {
    super();
    this.#tokens = tokens;
  }

  create(text: string) {
    const id = this.allDocuments.length;
    const uri = `file:///test-${id}.css`;
    const textDocument = {
      uri,
      languageId: "css",
      version: 1,
      text,
      /** Get the first position of the string in the document */
      positionOf: (
        string: string,
        position: "start" | "end" = "start",
      ): Position => {
        const text = this.getText(uri);
        // get the position of the string in doc
        const rows = text.split("\n");
        const line = rows.findIndex((line) => line.includes(string));
        let character = rows[line].indexOf(string);
        if (position === "end") {
          character += string.length;
        }
        return { line, character };
      },
      /** Get the first range of the string in the document */
      rangeOf: (string: string): Range => {
        const text = this.getText(uri);
        // get the range of the string in doc
        const rows = text.split("\n");
        const line = rows.findIndex((line) => line.includes(string));
        const character = rows[line].indexOf(string);
        return {
          start: { line, character },
          end: { line, character: character + string.length },
        };
      },
    };
    this.onDidOpen({ textDocument }, {
      documents: this,
      tokens: this.#tokens,
    });
    return textDocument;
  }

  tearDown() {
    for (const doc of this.allDocuments) {
      this.onDidClose({ textDocument: { uri: doc.uri } }, {
        documents: this,
        tokens: this.#tokens,
      });
    }
  }
}

/**
 * Test TokenMap for managing design tokens.
 */
class TestTokens extends TokenMap {
  #originalTokens: Record<string, Token>;
  #prefix: string;
  constructor(
    tokens = testTokens,
    prefix = "token",
  ) {
    super();
    this.#originalTokens = tokens;
    this.#prefix = prefix;
    this.reset();
  }

  reset() {
    this.clear();
    this.populateFromDtcg(
      this.#originalTokens,
      {
        prefix: this.#prefix,
        path: new URL("../test/tokens.json", import.meta.url).href,
      },
    );
  }

  override get(key: string) {
    return super.get(key.replace(/^-+/, ""));
  }

  override has(key: string) {
    return super.has(key.replace(/^-+/, ""));
  }
}

/**
 * Test LSP Client for interacting with the language server.
 *
 * This class provides methods to send messages to the server and handle responses.
 * It also manages the server process and handles errors.
 */
class TestLspClient {
  #lastId = 1;

  constructor(private server: Deno.ChildProcess) {}

  async #readMessage<M extends SupportedMethod>(): Promise<ResponseFor<M>> {
    const reader = this.server.stdout.getReader();
    const decoder = new TextDecoder();
    let buffer = "";

    try {
      while (true) {
        // Timeout in case server never responds
        const { value, done } = await reader.read();
        if (done) {
          // @ts-expect-error: fine
          return null; // No more data available, end of stream
        }

        buffer += decoder.decode(value, { stream: true });

        // Check for Content-Length header and parse it
        const contentLengthMatch = buffer.match(
          /^Content-Length: (\d+)\r\n\r\n/,
        );
        if (contentLengthMatch) {
          const contentLength = parseInt(contentLengthMatch[1], 10);
          const headerLength = contentLengthMatch[0].length;

          // Ensure we have received the entire message body
          if (buffer.length >= headerLength + contentLength) {
            const message = buffer.slice(
              headerLength,
              headerLength + contentLength,
            );
            return JSON.parse(message); // Return the complete message
          }
        }
      }
    } catch (e) {
      //handle any errors by checking stderr then rethrowing
      let stderr;

      try {
        stderr = this.server.stderr.getReader();
        const { value } = await stderr.read();
        console.error(decoder.decode(value, { stream: true }));
      } catch (e) {
        console.log("When logging stderr:", e);
      } finally {
        stderr?.releaseLock();
      }

      throw e;
    } finally {
      reader.releaseLock();
    }
  }

  /**
   * Send a message to the LSP server and wait for a response.
   *
   * @param message - The message to send to the server.
   * @returns The response from the server.
   */
  public async sendMessage<M extends SupportedMethod>(
    message: { method: M } & Omit<RequestTypeForMethod<M>, "jsonrpc" | "id">,
  ): Promise<ResponseFor<M> | null> {
    this.sendNotification(message, this.#lastId++);
    try {
      const resp = await this.#readMessage<M>();
      return resp as ResponseFor<M>;
    } catch (error) {
      console.error(error);
      this.server.kill();
      return null;
    }
  }

  /**
   * Send a notification to the LSP server.
   *
   * @param message - The message to send to the server.
   * @param id - Optional ID for the message.
   * @returns A promise that resolves when the message is sent.
   */
  public async sendNotification(message: object, id?: number) {
    const writer = this.server.stdin.getWriter();
    try {
      const bundle = { jsonrpc: "2.0", id, ...message };
      if (!id) delete bundle.id;
      const pkg = JSON.stringify(bundle);
      const encoder = new TextEncoder();
      const contentLength = pkg.length;
      const formattedMessage = `Content-Length: ${contentLength}\r\n\r\n${pkg}`;
      await writer.write(encoder.encode(formattedMessage));
    } finally {
      writer.releaseLock();
    }
  }

  /**
   * Close the LSP server and release resources.
   *
   * @returns A promise that resolves when the server is closed.
   */
  public async close() {
    await this.server.stderr.cancel();
    await this.server.stdin.close();
    await this.server.stdout.cancel();
    this.server.kill();
    return await this.server.status;
  }
}

/**
 * Create a test context for the language server.
 */
export function createTestContext() {
  const tokens = new TestTokens();
  const documents = new TestDocuments(tokens);
  return { documents, tokens };
}

/**
 * Create a test LSP client for interacting with the language server.
 * Spawns a server process and sets up communication over stdio
 */
export function createTestLspClient() {
  const server = new Deno.Command(Deno.execPath(), {
    stdin: "piped",
    stdout: "piped",
    stderr: "piped",
    args: ["-A", "--quiet", "./src/main.ts"],
  }).spawn();

  const client = new TestLspClient(server);
  return client;
}
