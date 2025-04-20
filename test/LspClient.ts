import { RequestTypeForMethod, ResponseFor, SupportedMethod } from "#lsp";

// Utility function to send and receive LSP messages
export class TestLspClient {
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
        if (done)
          // @ts-expect-error: fine
          return null; // No more data available, end of stream

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
    message: { method: M } & Omit<RequestTypeForMethod<M>, 'jsonrpc'|'id'>,
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

