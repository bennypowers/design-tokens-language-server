import type { Token } from "style-dictionary";

import { getLightDarkValues } from "#css";

import { convertTokenData } from "style-dictionary/utils";

import { TokenFileSpec } from "#lsp";
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

  meta = new Map<Token, TokenFileSpec>();

  populateFromDtcg(dtcgTokens: Record<string, Token>, spec: TokenFileSpec) {
    const flat = convertTokenData(structuredClone(dtcgTokens), {
      output: "map",
      usesDtcg: true,
    });
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
        this.meta.set(token, spec);
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
  return [
    `# \`--${name.replace(/^--/, "")}\``,
    "",
    // TODO: convert DTCG types to CSS syntax
    // const type = $type ? ` *<\`${$type}\`>*` : "";
    `Type: \`${$type}\``,
    $description ?? "",
    "",
    "```css",
    format($value),
    "```",
  ].join("\n");
}

export const tokens = new Tokens();
