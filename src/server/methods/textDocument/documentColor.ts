import type {
  ColorInformation,
  DocumentColorParams,
} from "vscode-languageserver-protocol";

import { tokens } from "../../storage.ts";
import { documents, tsRangeToLspRange } from "../../css/documents.ts";

import Color from "npm:tinycolor2";
import { getLightDarkValues } from "../../css/values.ts";

const HEX_RE = /#(?<hex>.{3}|.{4}|.{6}|.{8})\b/g;

export function documentColor(params: DocumentColorParams): ColorInformation[] {
  return documents.queryVarCalls(params.textDocument.uri).flatMap(result => {
    const tokenCap = result.captures.find(cap => cap.name === 'tokenName');
    if (!tokenCap) return [];
    const tokenName = tokenCap.node.text;
    const token = tokens.get(tokenName);
    if (!token || token.$type !== "color") return [];
    const colors = [];
    const hexMatches = `${token.$value}`.match(HEX_RE);
    const [light, dark] = getLightDarkValues(token.$value);
    if (hexMatches)
      colors.push(...hexMatches);
    else if (light && dark)
      colors.push(light, dark);
    else
      colors.push(token.$value);
    return colors.flatMap(match => {
      const color = Color(match);
      const prgb = color.toPercentageRgb();
      return [{
        color: {
          red: parseInt(prgb.r) * 0.01,
          green: parseInt(prgb.g) * 0.01,
          blue: parseInt(prgb.b) * 0.01,
          alpha: color.getAlpha(),
        },
        range: tsRangeToLspRange(tokenCap.node),
      } satisfies ColorInformation];
    });
  });
}
