import type { Token } from "style-dictionary";

import { getLightDarkValues } from "#css";

import { convertTokenData } from "style-dictionary/utils";
import { TokenFile } from "#lsp";
import { Logger } from "#logger";

export class TokenMap extends Map<string, Token> {
  override get(key: string) {
    return super.get(key.replace(/^-+/, ""));
  }
  override has(key: string) {
    return super.has(key.replace(/^-+/, ""));
  }

  populateFromDtcg(dtcgTokens: Record<string, Token>, prefix?: string) {
    const flat = convertTokenData(structuredClone(dtcgTokens), {
      output: "map",
      usesDtcg: true,
    });
    for (const [key, token] of flat) {
      if (key) {
        const joined = key
          .replace(/^\{(.*)}$/, "$1")
          .split(".")
          .filter((x) => !["_", "@", "DEFAULT"].includes(x)) // hack for dtcg tokens-that-are-also-groups
          .join("-");
        const name = prefix ? `${prefix}-${joined}` : joined;
        this.set(name, token);
      }
    }
  }

  public async register(tokenFile: TokenFile) {
    let spec = typeof tokenFile === "string" ? tokenFile : tokenFile.path;
    const prefix = typeof tokenFile === "string" ? undefined : tokenFile.prefix;
    try {
      if (spec.startsWith("~")) {
        spec = spec.replace("~", Deno.env.get("HOME")!);
      } else if (spec.startsWith(".")) {
        spec = spec.replace(".", Deno.cwd());
      }

      const { default: json } = await import(spec, { with: { type: "json" } });
      this.populateFromDtcg(json, prefix);
      Logger.info`Registered Tokens File: ${spec}\n${
        Object.fromEntries(
          this.entries().map(([k, v]) => [k, v.$value]),
        )
      }\n`;
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

export const tokens = new TokenMap();
