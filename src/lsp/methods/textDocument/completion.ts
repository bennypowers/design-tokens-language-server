import type { Token } from "style-dictionary";
import * as LSP from "vscode-languageserver-protocol";

import { tokens } from "#tokens";

import { documents, tsRangeToLspRange } from "#css";

const matchesWord =
  (word: string | null) =>
    ([name]: [name: string, x: Token]): boolean =>
      !!word && !!name && name
        .replaceAll("-", "")
        .startsWith(word.replaceAll("-", ""));

function offset(pos: LSP.Position, offset: Partial<LSP.Position>): LSP.Position {
  return {
    line: pos.line + (offset.line ?? 0),
    character: pos.character + (offset.character ?? 0),
  };
}

function escapeCommas($value: string) {
  return $value.replaceAll(',', '\\,');
}

/**
 * Generates completion items for design tokens.
 *
 * @param params - The parameters for the completion request.
 * @returns A completion list or an array of completion items representing the design tokens that match the specified word.
 */
export function completion(params: LSP.CompletionParams): null | LSP.CompletionList | LSP.CompletionItem[] {
  const node = documents.getNodeAtPosition(params.textDocument.uri, offset(params.position, { character: -2 }));
  if (!node) return null;
  const range = tsRangeToLspRange(node);
  const items = tokens.entries().filter(matchesWord(node.text))
  .map(([name, { $value }]) => ({
    label: name,
    kind: LSP.CompletionItemKind.Snippet,
    ...(range ? {
      textEdit: {
        range,
        newText: `var(--${name}\${1|\\, ${escapeCommas($value)},|})$0`,
      }
    } : {
      insertText: `var(--${name}\${1|\\, ${escapeCommas($value)},|})$0`,
    })
  }) satisfies LSP.CompletionItem).toArray();
  return {
    isIncomplete: items.length === 0 || items.length < tokens.size,
    itemDefaults: {
      insertTextFormat: LSP.InsertTextFormat.Snippet,
      insertTextMode: LSP.InsertTextMode.asIs,
      editRange: range,
    },
    items
  }
}

