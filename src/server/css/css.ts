import type { DocumentUri, Position, Range } from "vscode-languageserver-protocol";
import { documentTextCache } from "../documents.ts";

import type { Node, HardNode } from "https://deno.land/x/deno_tree_sitter@0.2.8.5/tree_sitter.js"
import { parserFromWasm } from "https://deno.land/x/deno_tree_sitter@0.2.8.5/main.js"

const parser = await Promise.resolve(parserFromWasm('https://github.com/jeff-hykin/common_tree_sitter_languages/raw/refs/heads/master/main/css.wasm'));

export interface TSQueryCapture {
  name: string;
  node: Omit<HardNode, 'children'> & TSNodePosition & { children: HardNode[] };
}

export interface TSQueryResult {
  pattern: number;
  captures: TSQueryCapture[]
}

export interface TSNodePosition {
  endPosition: { row: number; column: number; };
  startPosition: { row: number; column: number; }
}

export interface SyntaxNode extends Node, TSNodePosition {
  type: string;
  text: string;
}

// const cssTreeCache = new Map<DocumentUri, Tree>;

function getCssTreeForDocument(uri: DocumentUri) {
  // TODO: get cached tree and update incrementally in textDocument/didChange
  const text = documentTextCache.get(uri);
  const tree = parser.parse(text);
  return tree;
}

export function queryCssDocument(
  uri: DocumentUri,
  query: string,
  options?: TSNodePosition,
) {
  const rootNode = getCssTreeForDocument(uri).rootNode as HardNode
  return rootNode.query(query, options ?? {}) as unknown as TSQueryResult[];
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

export function tsNodeIsInLspRange(node: TSNodePosition, range: Range): boolean {
  const inRows =
       node.startPosition.row >= range.start.line
    && node.endPosition.row <= range.end.line;
  const inCols =
       node.startPosition.column >= range.start.character
    && node.endPosition.column <= range.end.character;
  return (inRows && inCols);
}

export function lspRangeIsInTsNode(node: TSNodePosition, range: Range): boolean {
  const inRows =
       node.startPosition.row >= range.start.line
    && node.endPosition.row <= range.end.line;
  const inCols =
       node.startPosition.column <= range.start.character
    && node.endPosition.column >= range.end.character;
  return (inRows && inCols);
}

export function tsNodeToLspRange(node: Pick<SyntaxNode, 'startPosition'|'endPosition'>): Range {
  return {
    start: { line: node.startPosition.row, character: node.startPosition.column },
    end: { line: node.endPosition.row, character: node.endPosition.column },
  }
}
