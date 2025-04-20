import * as LSP from "vscode-languageserver-protocol";

import { tokens } from "#tokens";

import { documents, tsRangeToLspRange } from "#css";
import { Node } from "web-tree-sitter";

const matchesWord = (word: string | null) => (name: string): boolean =>
  !!word && !!name && name
    .replaceAll("-", "")
    .startsWith(word.replaceAll("-", ""));

function offset(
  pos: LSP.Position,
  offset: Partial<LSP.Position>,
): LSP.Position {
  return {
    line: pos.line + (offset.line ?? 0),
    character: pos.character + (offset.character ?? 0),
  };
}

function escapeCommas($value: string) {
  return $value.replaceAll(",", "\\,");
}

function getCompletionDependingOnNode(node: Node, name: string): string {
  switch (node.type) {
    case "identifier":
    case "property_name":
      return `--${name}: $0`;
    default: {
      const token = tokens.get(name)!;
      return `var(--${name}\${1|\\, ${escapeCommas(token.$value)},|})$0`;
    }
  }
}

function getEditOrEntry(
  node: Node,
  name: string,
  range: LSP.Range,
): Pick<LSP.CompletionItem, "insertText" | "textEdit"> {
  const insertText = getCompletionDependingOnNode(node, name);
  return (range
    ? { textEdit: { range, newText: insertText } }
    : { insertText });
}

/**
 * Generates completion items for design tokens.
 *
 * @param params - The parameters for the completion request.
 * @param docs - The documents manager to retrieve the document at the specified position.
 * @returns A completion list or an array of completion items representing the design tokens that match the specified word.
 */
export function completion(
  params: LSP.CompletionParams,
  docs = documents,
): null | LSP.CompletionList | LSP.CompletionItem[] {
  const document = docs.get(params.textDocument.uri);

  const node = document.getNodeAtPosition(
    offset(params.position, { character: -2 }),
  );

  if (
    !node ||
    node.type !== "identifier" &&
      !document.positionIsInNodeType(params.position, "block")
  ) {
    return null;
  }

  const range = tsRangeToLspRange(node);
  const items = tokens
    .keys()
    .filter(matchesWord(node.text))
    .map((name) =>
      ({
        label: name,
        kind: LSP.CompletionItemKind.Snippet,
        ...getEditOrEntry(node, name, range),
      }) satisfies LSP.CompletionItem
    ).toArray();

  return {
    items,
    isIncomplete: items.length === 0 || items.length < tokens.size,
    itemDefaults: {
      insertTextFormat: LSP.InsertTextFormat.Snippet,
      insertTextMode: LSP.InsertTextMode.asIs,
      editRange: range,
    },
  };
}
