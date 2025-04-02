import type { RequestMessage, ResponseMessage } from "vscode-languageserver-protocol";

export class Logger {
  static #path = '/tmp/dt_ls.log';

  static #stream: Deno.FsFile;

  static async #init() {
    if (!this.#stream) {
      await Deno.create(this.#path);
      this.#stream = await Deno.open(this.#path, { write: true });
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
