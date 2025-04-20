import { Token } from "style-dictionary";

export { TestDocuments } from "./TestDocuments.ts";
export { TestLspClient } from "./LspClient.ts";

import testTokens from "../test/tokens.json" with { type: "json" };

import { populateMap } from "#tokens";

export class TestTokens extends Map<string, Token> {
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
    populateMap(this.#originalTokens, this, this.#prefix);
  }

  override get(key: string) {
    return super.get(key.replace(/^-+/, ""));
  }
  override has(key: string) {
    return super.has(key.replace(/^-+/, ""));
  }
}
