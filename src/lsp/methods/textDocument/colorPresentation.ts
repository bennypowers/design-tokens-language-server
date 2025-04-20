import type { Token } from "style-dictionary";
import type {
  ColorPresentation,
  ColorPresentationParams,
} from "vscode-languageserver-protocol";
import { tokens } from "#tokens";

import Color from "tinycolor2";

function isMatchFor(color: Color.Instance) {
  return ([name, token]: [string, Token]): boolean => {
    try {
      return name.startsWith("--") &&
        token.$type === "color" &&
        Color(token.$value).toHex8String() === color.toHex8String();
    } catch {
      return false;
    }
  };
}

/**
 * Generates color presentations for design tokens.
 *
 * @param params - The parameters for the color presentation request.
 * @returns An array of color presentations representing the design tokens that match the specified color.
 */
export function colorPresentation(
  { color }: ColorPresentationParams,
): ColorPresentation[] {
  return tokens
    .entries()
    .filter(
      isMatchFor(
        Color({ r: color.red, g: color.green, b: color.blue, a: color.alpha }),
      ),
    )
    .map(([label]) => ({ label }))
    .toArray();
}
