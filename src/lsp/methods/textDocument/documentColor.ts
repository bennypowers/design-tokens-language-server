import type {
  ColorInformation,
  DocumentColorParams,
} from "vscode-languageserver-protocol";

import { getLightDarkValues, getVarCallArguments } from "#css";

import { cssColorToLspColor } from "#color";

import { DTLSContext } from "#lsp";
import { Logger } from "#logger";

/**
 * Regular expression to match hex color values.
 */
const HEX_RE = /#(?<hex>.{3}|.{4}|.{6}|.{8})\b/g;

/**
 * Given that the match can be a hex color, a css color name, or a var call,
 * and that if it's a var call, it can be to a known token, or to an unknown
 * custom property, we need to extract the color value from the match.
 * We can't return the var call as-is, because tinycolor can't parse it.
 * So we need to return the fallback value of the var call, which itself could be a var call or
 * any valid css color value.
 *
 * We also need to handle the case where the var call is a known token, in which case we can just
 * return the value of the token.
 */
function extractColor(match: string, context: DTLSContext): string {
  if (match.startsWith("var(")) {
    const { variable, fallback } = getVarCallArguments(match);
    if (context.tokens.has(variable)) {
      return extractColor(context.tokens.get(variable)!.$value, context);
    } else if (fallback) {
      return extractColor(fallback, context);
    }
  }
  return match;
}

/**
 * Generates color information for design tokens.
 *
 * @param params - The parameters for the document color request.
 * @param context - The context containing design tokens and documents.
 * @returns An array of color information representing the design tokens found in the specified document.
 */
export function documentColor(
  params: DocumentColorParams,
  context: DTLSContext,
): ColorInformation[] {
  const doc = context.documents.get(params.textDocument.uri);
  if (doc.language === "css") {
    return doc.varCalls.flatMap((call) => {
      const token = call.token.token;

      if (!token || token.$type !== "color") {
        return [];
      }
      const colors = [];
      const hexMatches = `${token.$value}`.match(HEX_RE);
      const [light, dark] = getLightDarkValues(token.$value);
      if (light && dark) {
        colors.push(light, dark);
      } else if (hexMatches) {
        colors.push(...hexMatches);
      } else {
        colors.push(token.$value);
      }
      if (call.token.name === "--rh-color-interactive-primary-default") {
        Logger
          .debug`Found --rh-color-interactive-primary-default ${call}, ${colors}`;
      }
      return colors.flatMap((match) => {
        const color = extractColor(match, context);
        return [
          {
            color: cssColorToLspColor(color),
            range: call.token.range,
          } satisfies ColorInformation,
        ];
      });
    });
  }
  return [];
}
