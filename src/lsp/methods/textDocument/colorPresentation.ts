import type { Token } from "style-dictionary";
import type {
  ColorPresentation,
  ColorPresentationParams,
} from "vscode-languageserver-protocol";
import { tokens } from "#tokens";

import Color from "tinycolor2";
import { lspColorToTinyColor } from "../../../css/color.ts";

/**
 * Generates color presentations for design tokens.
 *
 * @param params - The parameters for the color presentation request.
 * @returns An array of color presentations representing the design tokens that match the specified color.
 */
export function colorPresentation(
  { color }: ColorPresentationParams,
): ColorPresentation[] {
  const instance = lspColorToTinyColor(color);
  return tokens
    .entries()
    .filter(([, token]: [string, Token]): boolean => {
      try {
        return token.$type === "color" &&
          Color(token.$value).toHex8String() === instance.toHex8String();
      } catch {
        return false;
      }
    })
    .map(([label]) => ({ label }))
    .toArray();
}
