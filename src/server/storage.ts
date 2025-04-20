import type { Token } from "style-dictionary";

import { convertTokenData } from "style-dictionary/utils";

export const tokens = new Map<string, Token>();

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
      tokens.set(`--${name}`, token);
    }
  }
}
