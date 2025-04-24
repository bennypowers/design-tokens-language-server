import type { Node } from "web-tree-sitter";
import type { DTLSContext } from "#lsp";
import * as LSP from "vscode-languageserver-protocol";
import { tsRangeToLspRange } from "#css";

interface CompletionArgs {
  node: Node;
  name: string;
  range: LSP.Range;
  tokens: DTLSContext["tokens"];
}

const matchesWord = (word: string | null) => (name: string): boolean =>
  !!word && !!name && name
    .replaceAll("-", "")
    .startsWith(word.replaceAll("-", ""));

function escapeCommas($value: string) {
  if (typeof $value !== "string") {
    return $value;
  } else {
    return $value.replaceAll(",", "\\,");
  }
}

function getCompletionDependingOnNode(args: CompletionArgs): string {
  const { node, name, tokens } = args;
  switch (node.type) {
    case "identifier":
    case "property_name":
      return `--${name}: $0`;
    default: {
      const token = tokens.get(name)!;
      const value = Array.isArray(token.$value)
        ? token.$value.join(", ")
        : token.$value;
      return `var(--${name}\${1|\\, ${escapeCommas(value)},|})$0`;
    }
  }
}

function getEditOrEntry(args: {
  node: Node;
  name: string;
  range: LSP.Range;
  tokens: DTLSContext["tokens"];
}): Pick<LSP.CompletionItem, "insertText" | "textEdit"> {
  const { range } = args;
  const insertText = getCompletionDependingOnNode(args);
  return { textEdit: { range, newText: insertText } };
}

/**
 * Generates completion items for design tokens.
 *
 * @param params - The parameters for the completion request.
 * @param context - The context containing the design tokens and documents.
 * @returns A completion list or an array of completion items representing the design tokens that match the specified word.
 */
export function completion(
  params: LSP.CompletionParams,
  { documents, tokens }: DTLSContext,
): null | LSP.CompletionList {
  const document = documents.get(params.textDocument.uri);

  if (document.language === "css") {
    const node = document.getNodeAtPosition(params.position, { character: -2 });

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
          ...getEditOrEntry({ node, name, range, tokens }),
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
  return null;
}
