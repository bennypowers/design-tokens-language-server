import type { Token } from "style-dictionary";
import { convertTokenData } from "style-dictionary/utils";

const tokens = new Map<string, Token>();

export async function register(spec: string) {
  if (spec.startsWith("~")) spec = spec.replace("~", Deno.env.get("HOME")!);
  const { default: json } = await import(spec, { with: { type: "json" } });
  const flat = convertTokenData(json, { output: "array", usesDtcg: true });
  for (const token of flat) {
    if (token.name) tokens.set(token.name, token);
  }
}

export function get(name: string): Token | null {
  return tokens.get(name) ?? null;
}

export function all() {
  return tokens.values();
}
