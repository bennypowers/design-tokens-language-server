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

export class Tokens extends Map<string, Token> {
  override get(key?: string) {
    if (key === undefined) {
      return undefined;
    }
    return super.get(key.replace(/^-+/, ""));
  }

  override has(key?: string) {
    if (key === undefined) {
      return false;
    }
    return super.has(key.replace(/^-+/, ""));
  }

  #dtcg: Record<string, Token> = {};
  specs = new Map<Token, TokenFileSpec>();

  resolve(reference: string) {
    return resolveReferences(reference, this.#dtcg, {
      usesDtcg: true,
    }) as string | number | Record<string, Token>;
  }

  populateFromDtcg(dtcgTokens: Record<string, Token>, spec: TokenFileSpec) {
    this.#dtcg = dtcgTokens;
    const flat = convertTokenData(
      typeDtcgDelegate(structuredClone(dtcgTokens)),
      {
        output: "map",
        usesDtcg: true,
      },
    );
    // hack for dtcg tokens-that-are-also-groups
    const groupMarkers = new Set(spec.groupMarkers ?? ["_", "@", "DEFAULT"]);
    for (const [key, token] of flat) {
      if (key) {
        const joined = key
          .replace(/^\{(.*)}$/, "$1")
          .split(".")
          .filter((x) => !groupMarkers.has(x))
          .join("-");
        const name = spec.prefix ? `${spec.prefix}-${joined}` : joined;
        if (usesReferences(token.$value)) {
          token.$value = resolveReferences(token.$value, dtcgTokens, {
            usesDtcg: true,
          });
        }
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
