import type { Position, Range } from "vscode-languageserver-protocol";
import { documentTextCache } from "../documents.ts";
import { Logger } from "../logger.ts";

interface CSSWord {
  word: string;
  range?: Range;
}

export function getCSSWordAtPosition(uri: string, position: Position): CSSWord {
  const text = documentTextCache.get(uri) ?? null;
  const line = text?.split('\n')[position.line] ?? null;
  const left = line?.substring(0, position.character);
  const right = line?.substring(position.character);
  // TODO: suck less
  const word = `${left?.replace(/^.+ /g, "")}${right?.replace(/ .+$/g, "")}`.replace(/var\(([^)]*)\)/, '$1').replace(';', '');
  const range: Range | undefined = line && left && right && {
    start: { ...position, character: line?.indexOf(left??'') ?? 0 },
    end: { ...position }
  } || undefined;
  return { word, range };
}

export function getCSSWordUntilPosition(uri: string, position: Position): null | string {
  const text = documentTextCache.get(uri) ?? null;
  const line = text?.split('\n')[position.line] ?? null;
  const until = line?.slice(0, position.character + 1) ?? null;
  const word = until?.replace(/.*\s+(.*?)/, "$1") ?? null;
  Logger.write({ uri, text, line, until, word });
  return word;
}
