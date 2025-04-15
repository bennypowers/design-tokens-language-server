import type { ColorPresentation, ColorPresentationParams } from "vscode-languageserver-protocol";
import { tokens } from "../../storage.ts";

export function colorPresentation(params: ColorPresentationParams): ColorPresentation[] {
  params.color
  params.textDocument
  // const { start, end } = params.range
  // const text = documentTextCache.get(params.textDocument.uri);
  // // QUESTION: multiline?
  // if (!text || start.line !== end.line)
  //   return [];
  // const word = text.split('\n')[start.line].substring(start.character, end.character);
  // const token = get(word.replace(/^--/, ''))
  // if (token) {}
  return tokens.entries().map(function ([name]) {
    return {
      label: `--${name}`,
    }
  }).toArray();
}
