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
  const flat = convertTokenData(json, { output: "array", usesDtcg: true });
  for (const flattened of flat) {
    const { key, ...token } = flattened;
    if (key) {
      const joined = key.replace(/^{(.*)}$/, '$1').replaceAll('.', '-');
      const name = tokenFile.prefix ? `${tokenFile.prefix}-${joined}` : joined;
      tokens.set(name, token);
      tokens.set(`--${name}`, token);
    }
  }
}

export function get(name: string): Token | null {
  return tokens.get(name) ?? null;
}

export function all() {
  return tokens.values();
}
