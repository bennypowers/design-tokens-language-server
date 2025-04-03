import {
  MessageType,
  type RequestMessage,
  type ResponseMessage,
} from "vscode-languageserver-protocol";

import * as path from "jsr:@std/path";

function isRequestMessage(message: unknown): message is RequestMessage {
  return !!message
    && typeof message === 'object'
    && ('id' in message)
    && ('method' in message)
}

function isResponseMessage(message: unknown): message is ResponseMessage {
  return !!message
    && typeof message === 'object'
    && ('id' in message)
    && !('method' in message)
}

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

  static async #write(message: unknown) {
    const kind = isRequestMessage(message) ? 'RECV' : isResponseMessage(message) ? 'SEND' : 'DBUG';
    await this.#init();
    const prefix = `\r\n\r\n// [dtls][${kind}][${Temporal.Now.plainDateTimeISO()}]\r\n`;
    const stringified = typeof message === 'object' ? JSON.stringify(message) : `${message}`
    this.#stream.write(this.#encoder.encode(`${prefix}${stringified}`));
  }

  static #report(message: string) {
    const messageLength = this.#encoder.encode(message).byteLength;
    const payload = `Content-Length: ${messageLength}\r\n\r\n${message}`;
    Deno.stdout.write(this.#encoder.encode(payload));
  }

  static logMessage(message: string|object, type: MessageType = MessageType.Log) {
    message = typeof message === 'string' ? message : JSON.stringify(message);
    this.#write(message);
    if (type === MessageType.Error)
      this.showMessage(message, type);
    const rpcmessage = JSON.stringify({ method: "window/logMessage", params: { message, type } });
    this.#report(rpcmessage);
  }

  static showMessage(message: string|object, type: MessageType = MessageType.Log) {
    const rpcmessage = JSON.stringify({ method: "window/showMessage", params: { message, type } });
    this.#report(rpcmessage);
  }

  static debug(message: string|object) { this.logMessage(message, MessageType.Debug); }
  static info(message: string|object) { this.logMessage(message, MessageType.Info); }
  static log(message: string|object) { this.logMessage(message, MessageType.Log); }
  static warn(message: string|object) { this.logMessage(message, MessageType.Warning); }
  static error(message: string|object) { this.logMessage(message, MessageType.Error); }
}
