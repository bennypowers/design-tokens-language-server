import { Parser, Language, Query } from "web-tree-sitter";
import type { QueryCapture, Node, Tree } from "web-tree-sitter";
import { readAll } from 'jsr:@std/io/read-all';
import { zip } from 'jsr:@std/collections/zip';

type TsRange = Pick<Node, 'startPosition'|'endPosition'>;

import {
  Diagnostic,
  DiagnosticSeverity,
  DidChangeTextDocumentParams,
  DidCloseTextDocumentParams,
  DidOpenTextDocumentParams,
  DocumentUri,
  Position,
  Range,
  TextDocumentContentChangeEvent,
} from "vscode-languageserver-protocol";

import {
  LightDarkValuesQuery,
  VarCall,
  VarCallWithFallback,
} from "./tree-sitter/queries.ts";

import { FullTextDocument } from "./textDocument.ts";
import { DTLSErrorCodes } from "../lsp/methods/textDocument/diagnostic.ts";

import { tokens } from "#tokens";

const f = await Deno.open(new URL("./tree-sitter/tree-sitter-css.wasm", import.meta.url))
const grammar = await readAll(f);

await Parser.init();
const parser = new Parser();
const Css = await Language.load(grammar);
parser.setLanguage(Css);

export interface TSNodePosition {
  endPosition: { row: number; column: number };
  startPosition: { row: number; column: number };
}

export function getLightDarkValues(value: string) {
  const tree = parser.parse(`a{b:${value}}`);
  if (!tree)
    return [];
  const query = new Query(tree.language, LightDarkValuesQuery);
  const captures = query.captures(tree.rootNode);
  const lightNode = captures.find(cap => cap.name === 'lightValue');
  const darkNode = captures.find(cap => cap.name === 'darkValue');
  return [lightNode?.node.text, darkNode?.node.text];
}

export function tsRangeToLspRange(node: TsRange|Node): Range {
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

export function captureIsTokenName(cap: QueryCapture) {
  return cap.name === "tokenName" &&
    tokens.has(cap.node.text.replace(/^--/, ""));
}

export function captureIsTokenCall(cap: QueryCapture) {
  return cap.name === "call" && !!cap.node.children
    .find((child) => child?.type === "arguments")
    ?.children
    .some((child) =>
      child?.type === "plain_value" &&
      tokens.has(child?.text.replace(/^--/, ""))
    );
}

class ENODOCError extends Error {
  constructor(public uri: DocumentUri) {
    super(`ENOENT: no CssDocument found for ${uri}`);
  }
}

class CssDocument extends FullTextDocument {
  #tree: Tree | null;

  diagnostics: Diagnostic[];

  constructor(uri: string, languageId: string, version: number, text: string) {
    super(uri, languageId, version, text);
    this.#tree = parser.parse(text);
    this.diagnostics = this.#computeDiagnostics();
  }

  override update(changes: TextDocumentContentChangeEvent[], version: number) {
    const old = this.getText();
    super.update(changes, version);
    const newText = this.getText();
    const newRows = newText.split('\n');
    if (!this.#tree)
      return;
    const oldEndPosition = this.#tree.rootNode.endPosition;
    this.#tree.edit({
      startIndex: 0,
      oldEndIndex: old.length,
      newEndIndex: newText.length,
      startPosition: { row: 0, column: 0 },
      oldEndPosition,
      newEndPosition: { row: newRows.length - 1, column: newRows[newRows.length - 1].length - 1 },
    });
    this.#tree = parser.parse(newText, this.#tree);
    this.diagnostics = this.#computeDiagnostics();
  }

  query(query: string, options?: TSNodePosition) {
    if (!this.#tree)
      return [];
    const q = new Query(this.#tree.language, query);
    return q.captures(this.#tree.rootNode, { matchLimit: 65536, ...options });
  }

  getNodeAtPosition(position: Position): null | Node {
    return this.#tree?.rootNode.descendantForPosition({
      row: position.line,
      column: position.character,
    }) ?? null;
  }

  #computeDiagnostics() {
    const captures = this.query(VarCallWithFallback);
    const tokenNameCaps = captures.filter(x => x.name === 'tokenName');
    const fallbackCaps = captures.filter(x => x.name === 'fallback');
    return zip(tokenNameCaps, fallbackCaps).flatMap(([tokenNameCap, fallbackCap]) => {
      if (tokens.has(tokenNameCap.node.text)) {
        const tokenName = tokenNameCap.node.text;
        const fallback = fallbackCap.node.text;
        const token = tokens.get(tokenName)!;
        const valid = fallback === token.$value;
        if (!valid)
          return [{
            range: tsRangeToLspRange(fallbackCap.node),
            severity: DiagnosticSeverity.Error,
            message: `Token fallback does not match expected value: ${token.$value}`,
            code: DTLSErrorCodes.incorrectFallback,
            data: {
              tokenName
            }
          }]
      }
      return [];
    })
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

  getDiagnostics(uri: DocumentUri) {
    const doc = this.get(uri);
    return doc.diagnostics;
  }

  getVersion(uri: DocumentUri) {
    const doc = this.get(uri);
    return doc.version;
  }

  onDidOpen(params: DidOpenTextDocumentParams) {
    const { uri, languageId,  version, text } = params.textDocument;
    const doc = new CssDocument(uri, languageId, version, text);
    this.#map.set(params.textDocument.uri, doc);
  }

  onDidChange(params: DidChangeTextDocumentParams) {
    const { uri, version } = params.textDocument;
    const doc = this.get(uri);
    doc.update(params.contentChanges, version);
  }

  onDidClose(params: DidCloseTextDocumentParams) {
    this.#map.delete(params.textDocument.uri);
  }

  getText(uri: DocumentUri) {
    return this.get(uri).getText();
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
