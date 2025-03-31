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

let tokens: Token[];

function computeSnippets() {
  return Array.from(all());
}

export function completion({ params }: CompletionRequestMessage): null | CompletionList {
  tokens ??= computeSnippets();
  const word = getCSSWordAtPosition(params.textDocument.uri, params.position);
  if (!word) return null;
  const items: CompletionItem[] = tokens.filter(x =>
      x.name?.replaceAll('-','').startsWith(word.replaceAll('-','')))
    .map(token => ({
      label: token.name!.replaceAll('-', '')!,
      kind: CompletionItemKind.Snippet,
      insertText: `var(--${token.name})$0`,
      documentation: token.$description && {
        value: getTokenMarkdown(token),
        kind: MarkupKind.Markdown,
      },
    }) satisfies CompletionItem);
  return {
    items,
    isIncomplete: false,
    itemDefaults: {
      insertTextFormat: InsertTextFormat.Snippet,
    }
  }
}
