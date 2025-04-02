import type { ColorInformation, DocumentColorParams } from "vscode-languageserver-protocol";

import { documentTextCache } from "../../documents.ts";
import { get } from "../../storage.ts";

import Color from "npm:tinycolor2"

export function documentColor(params: DocumentColorParams): ColorInformation[] {
  const text = documentTextCache.get(params.textDocument.uri) ?? '';
  const lines = text.split('\n');
  return lines.flatMap((lineTxt, line) => {
    let match
    const idxs: [match: string, start: number, end: number][] = [];
    const VAR_RE = /var\(--(?<name>[^)]+)\)/g;
    // deno-lint-ignore no-cond-assign
    while (match = VAR_RE.exec(lineTxt))
      idxs.push([
        match.groups!.name,
        match.index,
        match.index + match.groups!.name.length,
      ])
    return idxs.flatMap(([match, start, end]) => {
      const token = get(match);
      if (!token || token.$type !== 'color')
        return [];
      else {
        const [hex] = `${token.$value}`.match(/#(.{3}|.{4}|.{6}|.{8})/) ?? [];
        const color = Color(hex!);
        const { r, g, b } = color.toPercentageRgb();
        const info: ColorInformation = {
          color: {
            red: parseInt(r) * .01,
            green: parseInt(g) * .01,
            blue: parseInt(b) * .01,
            alpha: color.getAlpha(),
          },
          range: {
            start: { line, character: start },
            end: { line, character: end },
          }
        };
        return [info];
      }
    })
  });

}
