import type * as LSP from 'vscode-languageserver-protocol';
import type { Token } from 'style-dictionary';

import * as YAML from 'yaml';

import { getLightDarkValues } from '#css';

import {
  convertTokenData,
  resolveReferences,
  typeDtcgDelegate,
  usesReferences,
} from 'style-dictionary/utils';

import { DTLSContext, TokenFileSpec } from '#lsp/lsp.ts';
import { Logger } from '#logger';
import { PreprocessedTokens } from 'style-dictionary';
import { deepMerge } from '@std/collections/deep-merge';
import { toFileUrl } from '@std/path';

interface DTLSExtensions {
  /** The CSS var name for this token */
  name: `--${string}`;
  /** The file spec for this token's defining file */
  spec: TokenFileSpec;
  /** The path to this token e.g. ['color', 'red'] */
  path: string[];
  /** In the event that this is a group token, the group marker to include at the end of the path */
  groupMarker?: string;
  /** The string which references this token from another token e.g. `{color.red}` */
  reference: `{${string}}`;
  /** URI to the document which defines this token */
  definitionUri: LSP.DocumentUri;
}

export type DTLSToken = Omit<Token, '$extensions'> & {
  $extensions: {
    designTokensLanguageServer: DTLSExtensions;
  };
};

export const DEFAULT_GROUP_MARKERS = [
  '_',
  '@',
  'DEFAULT',
];

export class Tokens extends Map<string, DTLSToken> {
  /**
   * token-color-red => --token-color-red
   * {color.red} => --color-red
   */
  #normalizeKey(key: string) {
    if (key.startsWith(`{`))
      return `--${key.replace(/{|}/g, '').split('.').join('-')}`;
    else
      return `--${key}`.replace(/^-{4}/, '--');
  }

  override get(key?: string) {
    if (key === undefined)
      return undefined;
    const token = super.get(this.#normalizeKey(key));
    if (token && usesReferences(token.$value)) {
      const resolved = this.resolveValue(token.$value);
      const $value = resolved ?? token.$value;
      return { ...token, $value };
    }
    return token;
  }

  override has(key?: string) {
    if (key === undefined)
      return false;
    return super.has(this.#normalizeKey(key));
  }

  #importedSpecs = new Set();
  #dtcg?: Token;

  specs = new Map<DTLSToken, TokenFileSpec>();

  resolveValue(reference: string) {
    try {
      return resolveReferences(reference, this.#dtcg as PreprocessedTokens, {
        usesDtcg: true,
      }) as string | number;
    } catch {
      return null;
    }
  }

  populateFromDtcg(
    dtcgTokens: Record<string, Token>,
    spec: TokenFileSpec,
    context: DTLSContext,
  ) {
    const incoming = convertTokenData(
      typeDtcgDelegate(structuredClone(dtcgTokens)),
      {
        output: 'array',
        usesDtcg: true,
      },
    );
    const previous = convertTokenData(this.#dtcg ?? {}, {
      output: 'array',
      usesDtcg: true,
    });
    this.#dtcg = convertTokenData([...previous, ...incoming], {
      output: 'object',
      usesDtcg: true,
    });
    // hack for dtcg tokens-that-are-also-groups
    const groupMarkers = new Set(spec.groupMarkers ?? DEFAULT_GROUP_MARKERS);
    const map = convertTokenData(this.#dtcg, {
      output: 'map',
      usesDtcg: true,
    });
    for (const [key, token] of map) {
      if (key) {
        const path = key
          .replace(/^\{(.*)}$/, '$1')
          .split('.')
          .filter((x) => !groupMarkers.has(x));
        const modified = this.#getDTLSToken(token, spec, path, context);
        this.specs.set(modified, spec);
        const normalizedKey = modified.$extensions.designTokensLanguageServer.name;
        if (!this.has(normalizedKey)) this.set(normalizedKey, modified);
      }
    }
    return incoming.length;
  }

  #getDTLSToken(
    token: Token,
    spec: TokenFileSpec,
    path: string[],
    context: DTLSContext,
  ): DTLSToken {
    const clone = structuredClone(token);
    const prefix = spec.prefix ??
      context.workspaces.getSpecForUri(
        toFileUrl(spec.path.replace('file://', '')).href,
      );
    const prefixedPath = [prefix, ...path].filter((x) => !!x);
    const name = `--${prefixedPath.join('-')}` as const;
    const reference = `{${path.join('.')}}` as const;
    const definitionUri = toFileUrl(spec.path.replace('file://', '')).href;
    const groupMarkers = spec.groupMarkers ?? DEFAULT_GROUP_MARKERS;
    const keyPath = token.key?.replace(/{|}/g, '').split('.');
    const terminator = keyPath?.at(-1);
    const groupMarker = terminator && groupMarkers.includes(terminator) ? terminator : undefined;
    const ext = { name, spec, path, reference, definitionUri, groupMarker };
    const existing = clone?.$extensions?.designTokensLanguageServer ?? {};
    return {
      ...clone,
      $extensions: {
        ...clone.$extensions ?? {},
        designTokensLanguageServer: deepMerge<Partial<DTLSExtensions>>(
          existing,
          ext,
        ),
      },
    };
  }

  async #importSpec(path: string) {
    const language = path.split('.').pop();
    // TODO: handle this with ctx.documents
    switch (language) {
      case 'yaml':
      case 'yml':
        return YAML.parse(await Deno.readTextFile(path));
      default:
        return await import(path, { with: { type: 'json' } })
          .then((m) => m.default);
    }
  }

  public async register(
    spec: TokenFileSpec,
    { force = false } = {},
    context: DTLSContext,
  ) {
    if (force || !this.#importedSpecs.has(spec.path)) {
      try {
        const tokens = await this.#importSpec(spec.path);
        const amt = this.populateFromDtcg(tokens, spec, context);
        this.#importedSpecs.add(spec.path);
        Logger.info`✍️ Registered ${amt} tokens`;
        Logger.info`  from: ${spec.path}`;
        if (spec.prefix)
          Logger.info`  with prefix ${spec.prefix}`;
      } catch (e) {
        Logger.error`Could not load tokens for ${spec}: ${e}`;
      }
    }
  }
}

export function format(value: string): string {
  if (value?.startsWith?.('light-dark\(') && value.split('\n').length === 1) {
    const [light, dark] = getLightDarkValues(value);
    return `color: light-dark(
  ${light},
  ${dark}
)`;
  } else {
    return value;
  }
}

export function getTokenMarkdown(token: DTLSToken) {
  const { $description, $value, $type, $extensions } = token;
  const { name } = $extensions.designTokensLanguageServer;
  return [
    `# \`${name}\``,
    '',
    // TODO: convert DTCG types to CSS syntax
    // const type = $type ? ` *<\`${$type}\`>*` : "";
    `Type: \`${$type}\``,
    $description,
    '',
    '```css',
    format($value),
    '```',
  ].filter((x) => x != null).join('\n');
}

export const tokens = new Tokens();
