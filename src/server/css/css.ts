import type { DocumentUri, Position, Range } from "vscode-languageserver-protocol";
import { documentTextCache } from "../documents.ts";

import type { Node, HardNode } from "https://deno.land/x/deno_tree_sitter@0.2.8.5/tree_sitter.js"
import { parserFromWasm } from "https://deno.land/x/deno_tree_sitter@0.2.8.5/main.js"

const parser = await Promise.resolve(parserFromWasm('https://github.com/jeff-hykin/common_tree_sitter_languages/raw/refs/heads/master/main/css.wasm'));

interface TSQueryCapture {
  name: string;
  node: Omit<ReturnType<Node['toJSON']> & { hasChildren: boolean }, 'text'|'rootLeadingWhitespace'|'fieldNames'>;
}

interface TSQueryResult {
  pattern: number;
  captures: TSQueryCapture[]
}

export interface SyntaxNode extends HardNode {
  startPosition: HardNode['endPosition'];
}

// const cssTreeCache = new Map<DocumentUri, Tree>;

function getCssTreeForDocument(uri: DocumentUri) {
  // TODO: get cached tree and update incrementally in textDocument/didChange
  const text = documentTextCache.get(uri);
  const tree = parser.parse(text);
  return tree;
}

export function queryCssDocument(uri: DocumentUri, query: string) {
  const rootNode = getCssTreeForDocument(uri).rootNode as SyntaxNode
  return rootNode.query(query, {}) as unknown as TSQueryResult[];
}

export function getCssSyntaxNodeAtPosition(uri: DocumentUri, position: Position): null | SyntaxNode {
  return getCssTreeForDocument(uri).rootNode.descendantForPosition({
    row: position.line,
    column: position.character,
  })
}

export function tsNodeToRange(node: SyntaxNode): Range {
  return {
    start: { line: node.startPosition.row, character: node.startPosition.column },
    end: { line: node.endPosition.row, character: node.endPosition.column },
  };
}

