import type { Token } from "style-dictionary";

import { getLightDarkValues } from "#css";

import {
  convertTokenData,
  resolveReferences,
  typeDtcgDelegate,
  usesReferences,
} from "style-dictionary/utils";

import { TokenFileSpec } from "#lsp/lsp.ts";
import { Logger } from "#logger";
import { PreprocessedTokens } from "style-dictionary";

export class Tokens extends Map<string, Token> {
  override get(key?: string) {
    if (key === undefined) {
      return undefined;
    }
    const token = super.get(key.replace(/^-+/, ""));
    if (token && usesReferences(token.$value)) {
      return { ...token, $value: this.resolveValue(token.$value) };
    }
    return token;
  }

  override has(key?: string) {
    if (key === undefined) {
      return false;
    }
    return super.has(key.replace(/^-+/, ""));
  }

  #dtcg?: Token;
  specs = new Map<Token, TokenFileSpec>();

  resolveValue(reference: string) {
    return resolveReferences(reference, this.#dtcg as PreprocessedTokens, {
      usesDtcg: true,
    }) as
      | string
      | number
      | Record<string, Token>;
  }

  populateFromDtcg(dtcgTokens: Record<string, Token>, spec: TokenFileSpec) {
    const incoming = convertTokenData(
      typeDtcgDelegate(structuredClone(dtcgTokens)),
      {
        output: "array",
        usesDtcg: true,
      },
    );
    const previous = convertTokenData(this.#dtcg ?? {}, {
      output: "array",
      usesDtcg: true,
    });
    this.#dtcg = convertTokenData([...previous, ...incoming], {
      output: "object",
      usesDtcg: true,
    });
    // hack for dtcg tokens-that-are-also-groups
    const groupMarkers = new Set(spec.groupMarkers ?? ["_", "@", "DEFAULT"]);
    const map = convertTokenData(this.#dtcg, {
      output: "map",
      usesDtcg: true,
    });
    for (
      const [key, token] of map
    ) {
      if (key) {
        const joined = key
          .replace(/^\{(.*)}$/, "$1")
          .split(".")
          .filter((x) => !groupMarkers.has(x))
          .join("-");
        const name = spec.prefix ? `${spec.prefix}-${joined}` : joined;
        this.specs.set(token, spec);
        this.set(name, token);
      }
    }
  }

  public async register(spec: TokenFileSpec) {
    try {
      const { default: json } = await import(spec.path, {
        with: { type: "json" },
      });
      this.populateFromDtcg(json, spec);
      Logger
        .info`✍️ Registered ${this.size} tokens with prefix ${spec.prefix} from: ${spec.path}`;
    } catch {
      Logger.error`Could not load tokens for ${spec}`;
    }
  }
}

export function format(value: string): string {
  if (value?.startsWith?.("light-dark\(") && value.split("\n").length === 1) {
    const [light, dark] = getLightDarkValues(value);
    return `color: light-dark(
  ${light},
  ${dark}
)`;
  } else {
    return value;
  }
}

export function getTokenMarkdown(
  name: string,
  { $description, $value, $type }: Token,
) {
  const fancyName = name.startsWith("{")
    ? name.replace(/^\{(.*)}$/, "$1")
    : `--${name.replace(/^--/, "")}`;
  return [
    `# \`${fancyName}\``,
    "",
    // TODO: convert DTCG types to CSS syntax
    // const type = $type ? ` *<\`${$type}\`>*` : "";
    `Type: \`${$type}\``,
    $description,
    "",
    "```css",
    format($value),
    "```",
  ].filter((x) => x != null).join("\n");
}

export const tokens = new Tokens();
