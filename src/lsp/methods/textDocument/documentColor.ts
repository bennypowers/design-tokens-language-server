import type {
  ColorInformation,
  DocumentColorParams,
} from "vscode-languageserver-protocol";

import { tokens } from "#tokens";

import { documents, getLightDarkValues, tsRangeToLspRange } from "#css";

import { cssColorToLspColor } from "../../../css/color.ts";

/**
 * Regular expression to match hex color values.
 */
const HEX_RE = /#(?<hex>.{3}|.{4}|.{6}|.{8})\b/g;

/**
 * Generates color information for design tokens.
 *
 * @param params - The parameters for the document color request.
 * @returns An array of color information representing the design tokens found in the specified document.
 */
export function documentColor(params: DocumentColorParams): ColorInformation[] {
  return documents.queryVarCalls(params.textDocument.uri)
    .flatMap((cap) => {
      if (cap.name !== "tokenName") {
        return [];
      }
      const tokenName = cap.node.text;
      const token = tokens.get(tokenName.replace(/^--/, ""));
      if (!token || token.$type !== "color") {
        return [];
      }
      const colors = [];
      const hexMatches = `${token.$value}`.match(HEX_RE);
      const [light, dark] = getLightDarkValues(token.$value);
      if (hexMatches) {
        colors.push(...hexMatches);
      } else if (light && dark) {
        colors.push(light, dark);
      } else {
        colors.push(token.$value);
      }
      return colors.flatMap((match) => {
        return [
          {
            color: cssColorToLspColor(match),
            range: tsRangeToLspRange(cap.node),
          } satisfies ColorInformation,
        ];
      });
    });
}
