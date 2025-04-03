import {
  MessageType,
  type RequestMessage,
  type ResponseMessage,
} from "vscode-languageserver-protocol";

import * as path from "jsr:@std/path";

export class Logger {
  // static #path = `${Deno.env.get("XDG_STATE_HOME") ?? `${Deno.env.get("HOME")}/.local/state`}/design-tokens-language-server/dtls.log`;
  static #path = `/var/home/bennyp/.local/state/design-tokens-language-server/dtls.log`;

  static #stream: Deno.FsFile;

  static #encoder = new TextEncoder();

  static async #init() {
    if (!this.#stream) {
      await Deno.mkdir(path.dirname(this.#path), { recursive: true });
      await Deno.create(this.#path);
      this.#stream = await Deno.open(this.#path, { write: true });
    }
  }

  static debug(message: string|object) {
    this.logMessage(message, MessageType.Warning);
  }

  static info(message: string|object) {
    this.logMessage(message, MessageType.Warning);
  }

  static log(message: string|object) {
    this.logMessage(message, MessageType.Warning);
  }

  static warn(message: string|object) {
    this.logMessage(message, MessageType.Warning);
  }

  static error(message: string|object) {
    this.logMessage(message, MessageType.Error);
  }

  static logMessage(message: string|object, type: MessageType = MessageType.Log) {
    if (type === MessageType.Error) this.showMessage(message, type);
    const rpcmessage = JSON.stringify({ method: "window/logMessage", params: { message, type } });
    this.#payload(rpcmessage);
    this.write(message);
  }

  static showMessage(message: string|object, type: MessageType = MessageType.Log) {
    const rpcmessage = JSON.stringify({ method: "window/showMessage", params: { message, type } });
    this.#payload(rpcmessage);
  }

  static #payload(message: string) {
    const messageLength = this.#encoder.encode(message).byteLength;
    const payload = `Content-Length: ${messageLength}\r\n\r\n${message}`;
    Deno.stdout.write(this.#encoder.encode(payload));
  }

  static async write(message: RequestMessage | ResponseMessage | unknown) {
    await this.#init();
    const date = Temporal.Now.plainTimeISO();
    const prefix = `// [design-tokens-language-server][${date}]\r\n`;
    this.#stream.write(this.#encoder.encode(`${prefix}${typeof message === "object"
        ? JSON.stringify(message, null, 2)
        : (message as string)}`));
  }
}
