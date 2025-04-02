import {
  MessageType,
  type LogMessageParams,
  type RequestMessage,
  type ResponseMessage,
  type ShowMessageParams,
} from "vscode-languageserver-protocol";

import * as path from "jsr:@std/path";

const STATE =
  Deno.env.get("XDG_STATE_HOME") ?? `${Deno.env.get("HOME")}/.local/state`;

export class Logger {
  static #path = `${STATE}/design-tokens-language-server/dtls.log`;

  static #stream: Deno.FsFile;

  static async #init() {
    if (!this.#stream) {
      await Deno.mkdir(path.dirname(this.#path), { recursive: true });
      this.#stream = await Deno.open(this.#path, { write: true, create: true });
    }
  }

  static async write($message: RequestMessage | ResponseMessage | unknown) {
    await this.#init();
    const date = Temporal.Now.plainTimeISO();
    const prefix = `// [design-tokens-language-server][${date}]\r\n`;
    const message =
      typeof $message === "object"
        ? JSON.stringify($message, null, 2)
        : ($message as string);
    Deno.stdout.write(this.#encoder.encode(`${prefix}${message}`));
  }

  static logMessage(params: LogMessageParams) {
    const message = JSON.stringify({
      jsonrpc: "2a.0",
      method: "window/logMessage",
      params,
    });
    const messageLength = this.#encoder.encode(message).byteLength;
    const payload = `Content-Length: ${messageLength}\r\n\r\n${message}`;
    Deno.stdout.write(this.#encoder.encode(payload));
  }

  static error(m: string|object) {
    const message = typeof m === 'string' ? m : JSON.stringify(m);
    this.logMessage({ type: MessageType.Error, message });
    this.showMessage({ type: MessageType.Error, message });
  }

  static info(m: string|object) {
    const message = typeof m === 'string' ? m : JSON.stringify(m);
    this.logMessage({ type: MessageType.Info, message });
  }

  static log(m: string|object) {
    const message = typeof m === 'string' ? m : JSON.stringify(m);
    this.logMessage({ type: MessageType.Log, message });
  }

  static warn(m: string|object) {
    const message = typeof m === 'string' ? m : JSON.stringify(m);
    this.logMessage({ type: MessageType.Warning, message });
  }

  static showMessage(params: ShowMessageParams) {
    const message = JSON.stringify({
      jsonrpc: "2.0",
      method: "window/showMessage",
      params,
    });
    const messageLength = this.#encoder.encode(message).byteLength;
    const payload = `Content-Length: ${messageLength}\r\n\r\n${message}`;
    Deno.stdout.write(this.#encoder.encode(payload));
  }

  static #encoder = new TextEncoder();
}
