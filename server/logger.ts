import type { RequestMessage, ResponseMessage } from "vscode-languageserver-protocol";

import * as path from 'jsr:@std/path';

const STATE = Deno.env.get('XDG_STATE_HOME') ?? `${Deno.env.get('HOME')}/.local/state`;

export class Logger {
  static #path = `${STATE}/design-tokens-language-server/dtls.log`;

  static #stream: Deno.FsFile;

  static async #init() {
    if (!this.#stream) {
      await Deno.mkdir(path.dirname(this.#path), { recursive: true });
      this.#stream = await Deno.open(this.#path, { write: true, create: true });
    }
  }

  static async write(message: RequestMessage | ResponseMessage | unknown) {
    await this.#init();
    const date = Temporal.Now.plainTimeISO();
    const prefix = `// [design-tokens-language-server][${date}]\r\n`;
    if (typeof message === "object") {
      this.#stream.write(new TextEncoder().encode(`${prefix}${JSON.stringify(message, null, 2)}\r\n`));
    } else {
      this.#stream.write(new TextEncoder().encode(`${prefix}${message as string}\r\n`));
    }
  }
}
