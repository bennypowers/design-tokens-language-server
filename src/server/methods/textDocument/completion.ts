import type { Token } from "style-dictionary";
import {
  CompletionItem,
  CompletionItemKind,
  CompletionParams,
  CompletionList,
  InsertTextFormat,
  InsertTextMode,
  Position,
} from "vscode-languageserver-protocol";

import { tokens } from "../../storage.ts";

import { getCssSyntaxNodeAtPosition, tsNodeToRange } from "../../css/css.ts";
import { Logger } from "../../logger.ts";

const matchesWord =
  (word: string | null) =>
    ([name]: [name: string, x: Token]): boolean =>
      !!word && !!name && name
        .replaceAll("-", "")
        .startsWith(word.replaceAll("-", ""));

function offset(pos: Position, offset: Partial<Position>): Position {
  return {
    line: pos.line + (offset.line ?? 0),
    character: pos.character + (offset.character ?? 0),
  };
}

export async function completion(params: CompletionParams): Promise<null | CompletionList | CompletionItem[]> {
  const node = getCssSyntaxNodeAtPosition(params.textDocument.uri, offset(params.position, { character: -2 }));
  if (!node) return null;
  const range = tsNodeToRange(node);
  const items = tokens.entries().filter(matchesWord(node.text))
  .map(([name, { $value }]) => ({
    label: name,
    kind: 15 satisfies typeof CompletionItemKind.Snippet,
    ...(range ? {
      textEdit: {
        range,
        newText: `var(--${name}\${1|\\, ${$value},|})$0`,
      }
    } : {
      insertText: `var(--${name}\${1|\\, ${$value},|}):0`,
    })
  }) satisfies CompletionItem).toArray();
  return {
    // TODO: perf
    isIncomplete: items.length === 0 || items.length < tokens.size,
    itemDefaults: {
      insertTextFormat: InsertTextFormat.Snippet,
      insertTextMode: InsertTextMode.asIs,
      editRange: range,
    },
    items
  }
}
