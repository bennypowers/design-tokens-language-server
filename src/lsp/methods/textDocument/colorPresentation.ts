import type { DTLSContext } from "#lsp";

import type {
  ColorPresentation,
  ColorPresentationParams,
} from "vscode-languageserver-protocol";

import Color from "tinycolor2";

import { lspColorToTinyColor } from "#color";

/**
 * Generates color presentations for design tokens.
 *
 * @param params - The parameters for the color presentation request.
 * @param context - The context containing design tokens and other information.
 * @returns An array of color presentations representing the design tokens that match the specified color.
 */
export function colorPresentation(
  { color }: ColorPresentationParams,
  { tokens }: DTLSContext,
): ColorPresentation[] {
  const instance = lspColorToTinyColor(color);
  return tokens
    .entries()
    .filter(([, token]): boolean => {
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
