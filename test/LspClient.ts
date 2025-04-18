import * as LSP from "npm:vscode-languageserver-protocol";
import { SupportedMessage, SupportedNotificationMessage } from "../src/server/methods/initialize.ts";

type SupportedMethod = SupportedMessage['method'];

type ResultFor<M extends SupportedMethod> =
    M extends SupportedNotificationMessage['method'] ? null
  : M extends 'initialize' ? LSP.InitializeResult
  : M extends 'textDocument/hover' ? LSP.Hover
  : M extends 'textDocument/diagnostic' ? LSP.FullDocumentDiagnosticReport
  : never;

type ParamsFor<M extends SupportedMethod> =
    M extends 'initialize' ? LSP.InitializeParams
  : M extends 'initialized' ? LSP.InitializedParams
  : M extends 'textDocument/hover' ? LSP.HoverParams
  : M extends 'textDocument/diagnostic' ? LSP.DocumentDiagnosticParams
  : M extends 'textDocument/didOpen' ? LSP.DidOpenTextDocumentParams
  : M extends 'textDocument/didChange' ? LSP.DidChangeTextDocumentParams
  : M extends 'textDocument/didClose' ? LSP.DidCloseTextDocumentParams
  : never;

type RequestFor<M extends SupportedMethod> =
  Omit<LSP.RequestMessage, 'id'|'jsonrpc'|'method'> & {
     method: M;
     params: ParamsFor<M>
  };

type ResponseFor<M extends SupportedMethod> =
  null |
  M extends SupportedNotificationMessage['method'] ? null
: Omit<LSP.ResponseMessage, 'result'> & { result: ResultFor<M> };

// Utility function to send and receive LSP messages
export class LspClient {
  static NOTIFICATIONS = new Set([
    'textDocument/didOpen',
    'textDocument/didChange',
    'textDocument/didClose',
    '$/cancelRequest',
    '$/setTrace',
  ] as const satisfies SupportedNotificationMessage['method'][]);

  id = 1;

  constructor(private server: Deno.ChildProcess) {}

  // Function to read a complete message based on Content-Length
  async readMessage<M extends SupportedMethod>(): Promise<ResultFor<M>> {
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

  // Function to send a message to the LSP server
  async sendLspMessage<M extends SupportedMessage['method']>(
    message: RequestFor<M>,
  ): Promise<ResponseFor<M> | null> {
    this.sendNotification(message, this.id++);
    try {
      const resp = await this.readMessage();
      return resp as ResponseFor<M>;
    } catch (error) {
      console.error(error);
      this.server.kill();
      return null;
    }
  }

  // Function to send a message to the LSP server
  async sendNotification(message: object, id?: number) {
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

  async close() {
    await this.server.stderr.cancel();
    await this.server.stdin.close();
    await this.server.stdout.cancel();
    this.server.kill();
  }
}

