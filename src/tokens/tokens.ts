import type { Token } from "style-dictionary";

import * as YAML from "yaml";

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
      const resolved = this.resolveValue(token.$value);
      const $value = resolved ?? token.$value;
      return { ...token, $value };
    }
    return token;
  }

  override has(key?: string) {
    if (key === undefined) {
      return false;
    }
    return super.has(key.replace(/^-+/, ""));
  }

  #importedSpecs = new Set();
  #dtcg?: Token;

  specs = new Map<Token, TokenFileSpec>();

  resolveValue(reference: string) {
    try {
      return resolveReferences(reference, this.#dtcg as PreprocessedTokens, {
        usesDtcg: true,
      }) as string | number;
    } catch {
      return null;
    }
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
    return incoming.length;
  }

  async #importSpec(path: string) {
    const language = path.split(".").pop();
    // TODO: handle this with ctx.documents
    switch (language) {
      case "yaml":
      case "yml":
        return YAML.parse(await Deno.readTextFile(path));
      default:
        return await import(path, { with: { type: "json" } })
          .then((m) => m.default);
    }
  }

  public async register(spec: TokenFileSpec, { force = false } = {}) {
    if (force || !this.#importedSpecs.has(spec.path)) {
      try {
        const tokens = await this.#importSpec(spec.path);
        const amt = this.populateFromDtcg(tokens, spec);
        this.#importedSpecs.add(spec.path);
        Logger.info`✍️ Registered ${amt} tokens`;
        Logger.info`  from: ${spec.path}`;
        if (spec.prefix) {
          Logger.info`  with prefix ${spec.prefix}`;
        }
      } catch {
        Logger.error`Could not load tokens for ${spec}`;
      }
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
