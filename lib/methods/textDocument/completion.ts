import type { Token } from "style-dictionary";
import {
  type CompletionItem,
  type RequestMessage,
  CompletionParams,
  InsertTextMode,
  InsertTextFormat,
  CompletionItemKind,
  CompletionList,
  MarkupKind,
} from "npm:vscode-languageserver-protocol";

import { all } from "../../storage.ts";

import { getCSSWordAtPosition } from "../../css/css.ts";
import { getTokenMarkdown } from "../../token.ts";

export interface CompletionRequestMessage extends RequestMessage {
  params: CompletionParams;
}

let tokens: CompletionItem[];

function getAllSnippets() {
  return Array.from(all())
    .map((token, i, a) => ({
      label: token.name!,
      kind: CompletionItemKind.Snippet,
      insertText: `var(--${token.name})$0`,
      insertTextFormat: InsertTextFormat.Snippet,
      insertTextMode: InsertTextMode.asIs,
      sortText: (i + 1).toString().padStart(a.length.toString().length, '0'),
      documentation: token.$description && {
        value: getTokenMarkdown(token),
        kind: MarkupKind.Markdown,
      },
    }) satisfies CompletionItem)
}

const matchesWord = (word: string) =>
  (x: Token) =>
    x.name &&
    x.name.replaceAll('-','').startsWith(word.replaceAll('-',''))

export function completion(
  { params }: CompletionRequestMessage,
): null | CompletionList {
  tokens ??= getAllSnippets();
  const { word } = getCSSWordAtPosition(params.textDocument.uri, params.position);
  if (!word) return null;
  const items: CompletionItem[] = tokens.filter(matchesWord(word))
  return {
    items,
    isIncomplete: true,
    itemDefaults: {
      insertTextFormat: InsertTextFormat.Snippet,
      insertTextMode: InsertTextMode.asIs,
    }
  }
}
