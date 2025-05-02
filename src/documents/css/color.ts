import { Color } from 'vscode-languageserver-protocol';
import TinyColor from 'tinycolor2';

export function cssColorToLspColor(color: string): Color | null {
  const colorObj = TinyColor(color);
  if (colorObj.isValid()) {
    const prgb = colorObj.toPercentageRgb();
    return {
      red: parseInt(prgb.r) * 0.01,
      green: parseInt(prgb.g) * 0.01,
      blue: parseInt(prgb.b) * 0.01,
      alpha: colorObj.getAlpha(),
    };
  } else {
    return null;
  }
}

export function lspColorToTinyColor(color: Color): TinyColor.Instance {
  return new TinyColor({
    r: Math.round(color.red * 255),
    g: Math.round(color.green * 255),
    b: Math.round(color.blue * 255),
  }).setAlpha(color.alpha);
}
