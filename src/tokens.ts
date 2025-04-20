import type { Token } from "style-dictionary";

import { getLightDarkValues } from "#css";

import { convertTokenData } from "style-dictionary/utils";

class TokenMap extends Map<string, Token> {
  override get(key: string) {
    return super.get(key.replace(/^-+/, ''));
  }
  override has(key: string) {
    return super.has(key.replace(/^-+/, ''));
  }
}

export const tokens = new TokenMap();

export interface TokenFile {
  prefix?: string;
  path: string;
}

export async function register(tokenFile: TokenFile) {
  let spec = tokenFile.path;
  if (spec.startsWith("~"))
    spec = spec.replace("~", Deno.env.get("HOME")!);
  else if (spec.startsWith('.'))
    spec = spec.replace(".", Deno.cwd());

  const { default: json } = await import(spec, { with: { type: "json" } });
  const flat = convertTokenData(json, { output: "map", usesDtcg: true });
  for (const [key, token] of flat) {
    if (key) {
      const joined = key
        .replace(/^\{(.*)}$/, '$1')
        .split('.')
        .filter(x => !['_', '@', "DEFAULT"].includes(x)) // hack for dtcg tokens-that-are-also-groups
        .join('-');
      const name = tokenFile.prefix ? `${tokenFile.prefix}-${joined}` : joined;
      tokens.set(name, token);
    }
  }
}

function format(value: string): string {
  if (value?.startsWith?.("light-dark\(") && value.split("\n").length === 1) {
    const [light, dark] = getLightDarkValues(value);
    return `color: light-dark(
  ${light},
  ${dark}
)`;
  } else
    return value;
}

export function getTokenMarkdown(name: string, { $description, $value, $type }: Token) {
  return [
    `# \`--${name.replace(/^--/, '')}\``,
    '',
    // TODO: convert DTCG types to CSS syntax
    // const type = $type ? ` *<\`${$type}\`>*` : "";
    `Type: \`${$type}\``,
    $description ?? '',
    '',
    '```css',
    format($value),
    '```',
  ].join("\n");
}
