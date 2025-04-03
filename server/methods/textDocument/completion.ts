import type { Token } from "style-dictionary";
import type { CompletionItem, CompletionParams, CompletionList } from "vscode-languageserver-protocol";

import { all } from "../../storage.ts";

import { getCSSWordAtPosition } from "../../css/css.ts";
import { Logger } from "../../logger.ts";

const matchesWord = (word: string | null) => (x: Token): x is Token & { name: string } =>
  (!!word && !!x.name) && x.name.replaceAll("-", "").startsWith(word.replaceAll("-", ""));

export function completion(params: CompletionParams): null | CompletionList | CompletionItem[] {
  const { word } = getCSSWordAtPosition(params.textDocument.uri, params.position);
  try {
    return all().filter(matchesWord(word)).map(({ name }) => ({ label: name })).toArray();
  } catch (e) {
    Logger.error(`${e}`);
    return null
  }
}
