import type { ColorPresentation, ColorPresentationParams } from "vscode-languageserver-protocol";

export function colorPresentation(params: ColorPresentationParams): ColorPresentation[] {
  const { textDocument } = params
  return []
}
