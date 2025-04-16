import type {
  DidChangeTextDocumentParams,
  DidCloseTextDocumentParams,
  DocumentUri,
  Position,
  Range,
  TextDocumentContentChangeEvent,
} from "vscode-languageserver-protocol";

import type {
  HardNode,
  Node,
  Tree,
} from "https://deno.land/x/deno_tree_sitter@0.2.8.5/tree_sitter.js";
import { parserFromWasm } from "https://deno.land/x/deno_tree_sitter@0.2.8.5/main.js";
import { DidOpenTextDocumentParams } from "vscode-languageserver-protocol";
import { tokens } from "../storage.ts";
import { VarCall, VarCallWithFallback } from "./tree-sitter/queries.ts";
import { Logger } from "../logger.ts";

interface HardTree extends Tree {
  rootNode: HardNode;
}

export interface TSQueryCapture {
  name: string;
  node: Omit<HardNode, "children"> & TSNodePosition & { children: HardNode[] };
}

export interface TSQueryResult {
  pattern: number;
  captures: TSQueryCapture[];
}

export interface TSNodePosition {
  endPosition: { row: number; column: number };
  startPosition: { row: number; column: number };
}

export interface SyntaxNode extends Node, TSNodePosition {
  type: string;
  text: string;
}

type TsRange = Pick<SyntaxNode, "startPosition" | "endPosition">;

export const parser = await Promise.resolve(
  parserFromWasm(
    "https://github.com/jeff-hykin/common_tree_sitter_languages/raw/refs/heads/master/main/css.wasm",
  ),
);

export function tsRangeToLspRange(node: TsRange|HardNode): Range {
  return {
    start: {
      line: (node as TsRange).startPosition.row,
      character: (node as TsRange).startPosition.column,
    },
    end: {
      line: node.endPosition.row,
      character: node.endPosition.column,
    },
  };
}

export function lspRangeToTsRange(range: Range): TsRange {
  return {
    startPosition: {
      row: range.start.line,
      column: range.start.character,
    },
    endPosition: {
      row: range.end.line,
      column: range.end.character,
    },
  }
}

export function tsNodeIsInLspRange(node: TSNodePosition, range: Range): boolean {
  const inRows = node.startPosition.row >= range.start.line &&
    node.endPosition.row <= range.end.line;
  const inCols = node.startPosition.column >= range.start.character &&
    node.endPosition.column <= range.end.character;
  return (inRows && inCols);
}

export function lspRangeIsInTsNode(node: TSNodePosition, range: Range): boolean {
  const inRows = node.startPosition.row >= range.start.line &&
    node.endPosition.row <= range.end.line;
  const inCols = node.startPosition.column <= range.start.character &&
    node.endPosition.column >= range.end.character;
  return (inRows && inCols);
}

export function captureIsTokenName(cap: TSQueryCapture) {
  return cap.name === "tokenName" &&
    tokens.has(cap.node.text.replace(/^--/, ""));
}

export function captureIsTokenCall(cap: TSQueryCapture) {
  return cap.name === "call" && !!cap.node.children
    .find((child) => child.type === "arguments")
    ?.children
    .some((child) =>
      child.type === "plain_value" &&
      tokens.has(child.text.replace(/^--/, ""))
    );
}

class ENODOCError extends Error {
  constructor(public uri: DocumentUri) {
    super(`ENOENT: no CssDocument found for ${uri}`);
  }
}

class CssDocument {
  #tree: HardTree;

  constructor(
    public uri: string,
    public text: string,
    public version: number,
    public languageId: string,
  ) {
    this.#tree = parser.parse(text);
  }

  update(change: TextDocumentContentChangeEvent, version: number) {
    const old = this.text;
    const rows = old.split('\n');
    const oldEndPosition = this.#tree.rootNode.endPosition;
    if ('range' in change) {
      const startIndex = Iterator
        .from(rows)
        .take(change.range.start.line + 1)
        .reduce((sum, row, i) => i >= change.range.start.character ? sum : sum + row.length, 0)
      const oldEndIndex = Iterator
        .from(rows)
        .take(change.range.end.line + 1)
        .reduce((sum, row, i) => i >= change.range.end.character ? sum : sum + row.length, 0)
      if (!change.text)
        Logger.debug(`{change}`, { change });
      this.text = `${this.text.substring(0, startIndex)}${change.text}${this.text.substring(oldEndIndex, this.text.length)}`;
      const newRows = this.text.split('\n');
      this.#tree.edit({
        startIndex,
        oldEndIndex,
        newEndIndex: startIndex + change.text.length,
        ...lspRangeToTsRange(change.range),
        oldEndPosition,
        newEndPosition: { row: newRows.length-1, column: newRows.at(-1)!.length -1 },
      })
    } else {
      const newRows = change.text.split('\n');
      this.text = change.text;
      this.#tree.edit({
        startIndex: 0,
        oldEndIndex: old.length,
        newEndIndex: change.text.length,
        startPosition: { row: 0, column: 0 },
        oldEndPosition,
        newEndPosition: { row: newRows.length-1, column: newRows.at(-1)!.length -1 },
      })
    }
    this.version = version;
    this.#tree = parser.parse(this.text, this.#tree);
  }

  query(query: string, options?: TSNodePosition) {
    return this.#tree.rootNode.query(query, options ?? {}) as unknown as TSQueryResult[];
  }

  getNodeAtPosition(position: Position): null | SyntaxNode {
    return this.#tree.rootNode.descendantForPosition({
      row: position.line,
      column: position.character,
    });
  }
}

class Documents {
  #map = new Map<DocumentUri, CssDocument>();

  get(uri: DocumentUri) {
    const doc = this.#map.get(uri);
    if (!doc)
      throw new ENODOCError(uri);
    return doc;
  }

  onDidOpen(params: DidOpenTextDocumentParams) {
    this.#map.set(params.textDocument.uri, new CssDocument(
      params.textDocument.uri,
      params.textDocument.text,
      params.textDocument.version,
      params.textDocument.languageId,
    ))
  }

  onDidChange(params: DidChangeTextDocumentParams) {
    const { uri, version } = params.textDocument;
    const doc = this.get(uri);
    for (const change of params.contentChanges)
      doc.update(change, version);
  }

  onDidClose(params: DidCloseTextDocumentParams) {
    this.#map.delete(params.textDocument.uri);
  }

  getText(uri: DocumentUri) {
    return this.get(uri).text;
  }

  getNodeAtPosition(uri: DocumentUri, position: Position) {
    return this.get(uri).getNodeAtPosition(position);
  }

  queryVarCalls(uri: DocumentUri) {
    return this.get(uri).query(VarCall);
  }

  queryVarCallsWithFallback(uri: DocumentUri) {
    return this.get(uri).query(VarCallWithFallback);
  }
}

export const documents = new Documents();
