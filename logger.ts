import type { RequestMessage, ResponseMessage } from "./main.ts";

export class Logger {
  static #stream = Deno.openSync(Deno.makeTempFileSync());

  static write(message: RequestMessage | ResponseMessage | unknown) {
    if (typeof message === "object") {
      this.#stream.write(new TextEncoder().encode(JSON.stringify(message)));
    } else {
      this.#stream.write(new TextEncoder().encode(message as string));
    }
  }
}
