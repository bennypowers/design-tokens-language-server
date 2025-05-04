import type { Token } from "style-dictionary";
import type { DTLSContext } from "#lsp";

import type {
  ColorPresentation,
  ColorPresentationParams,
} from "vscode-languageserver-protocol";

import Color from "tinycolor2";

import { lspColorToTinyColor } from "#color";

function compareColors(
  token: Token,
  instance: Color.Instance,
): boolean {
  const c = Color(token.$value);
  return c.toHex8String() === instance.toHex8String();
}

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
    .flatMap(([label, t]) =>
      t.$type === "color" && compareColors(t, instance) ? [{ label }] : []
    )
    .toArray();
}
