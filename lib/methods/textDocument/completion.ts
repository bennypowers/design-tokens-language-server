import type { Token } from "style-dictionary";

import type {
  CompletionItem,
  RequestMessage ,
  CompletionParams,
  InsertTextMode,
  InsertTextFormat,
  CompletionItemKind,
  CompletionList,
  MarkupKind,
} from "vscode-languageserver-protocol";

import { all } from "../../storage.ts";

import { getCSSWordAtPosition } from "../../css/css.ts";
import { getTokenMarkdown } from "../../token.ts";
import { Logger } from "../../logger.ts";

export interface CompletionRequestMessage extends RequestMessage {
  params: CompletionParams;
}

let tokens: CompletionItem[];

function getAllSnippets() {
  return Array.from(all())
    .map((token, i, a) => ({
      label: token.name!,
      kind: 15 satisfies typeof CompletionItemKind.Snippet,
      insertText: `var(--${token.name}\${0:, ${token.$value}}):`,
      sortText: (i + 1).toString().padStart(a.length.toString().length, '0'),
      documentation: token.$description && {
        value: getTokenMarkdown(token),
        kind: 'markdown' satisfies typeof MarkupKind.Markdown,
      },
    }) satisfies CompletionItem)
}

const matchesWord = (word: string) =>
  (x: Token) =>
    x.name &&
    x.name.replaceAll('-','').startsWith(word.replaceAll('-',''))

export function completion({ params }: CompletionRequestMessage): null | CompletionList {
  tokens ??= getAllSnippets();
  const { word } = getCSSWordAtPosition(params.textDocument.uri, params.position);
  Logger.write(`completion word: ${word}`);
  if (!word) return null;
  const items: CompletionItem[] = tokens.filter(matchesWord(word))
  return {
    items,
    isIncomplete: true,
    itemDefaults: {
      insertTextFormat: 2 satisfies typeof InsertTextFormat.Snippet,
      insertTextMode: 1 satisfies typeof InsertTextMode.asIs,
    }
  }
}
