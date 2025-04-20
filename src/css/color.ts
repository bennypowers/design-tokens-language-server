import { Color } from "vscode-languageserver-protocol";
import TinyColor from "tinycolor2";

export function cssColorToLspColor(color: string): Color {
  const colorObj = TinyColor(color);
  const prgb = colorObj.toPercentageRgb();
  return {
    red: parseInt(prgb.r) * 0.01,
    green: parseInt(prgb.g) * 0.01,
    blue: parseInt(prgb.b) * 0.01,
    alpha: colorObj.getAlpha(),
  };
}
