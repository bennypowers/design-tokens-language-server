import type { ColorInformation, DocumentColorParams } from "vscode-languageserver-protocol";

import { documentTextCache } from "../../documents.ts";
import { get } from "../../storage.ts";

import Color from "npm:tinycolor2"
import { Logger } from "../../logger.ts";

interface Match {
  name: string;
  start: number;
  end: number;
}

const HEX_RE = /#(?<hex>.{3}|.{4}|.{6}|.{8})\b/g;

export function documentColor(params: DocumentColorParams): ColorInformation[] {
  const text = documentTextCache.get(params.textDocument.uri) ?? '';
  const lines = text.split('\n');
  return lines.flatMap((lineTxt, line) => {
    let match
    const idxs: Match[] = [];
    const VAR_RE = /var\(--(?<name>[^)]+)\)/g;
    // deno-lint-ignore no-cond-assign
    while (match = VAR_RE.exec(lineTxt)) {
      const { name } = match.groups!;
      const start = match.index + 4; // var(
      const end = start + match.groups!.name.length + 2; // --
      idxs.push({ name, start, end });
    }
    return idxs.flatMap(({ name, start, end }) => {
      const token = get(name);
      if (!token || token.$type !== 'color')
        return [];
      else {
        return (`${token.$value}`.match(HEX_RE) ?? []).map(hex => {
          Logger.write(`${token.name}: ${hex}`);
          const color = Color(hex);
          const prgb = color.toPercentageRgb();
          return {
            color: {
              red: parseInt(prgb.r) * .01,
              green: parseInt(prgb.g) * .01,
              blue: parseInt(prgb.b) * .01,
              alpha: color.getAlpha(),
            },
            range: {
              start: { line, character: start },
              end: { line, character: end },
            }
          } satisfies ColorInformation;
        });
      }
    })
  });

}
