import { Language, Parser, Query } from "web-tree-sitter";
import type { Node, Point, QueryCapture, Tree } from "web-tree-sitter";

import { readAll } from "jsr:@std/io/read-all";
import { zip } from "jsr:@std/collections/zip";

type TsRange = Pick<Node, "startPosition" | "endPosition">;

import * as LSP from "vscode-languageserver-protocol";

import {
  LightDarkValuesQuery,
  VarCall,
  VarCallWithFallback,
} from "./tree-sitter/queries.ts";

import { FullTextDocument } from "./textDocument.ts";
import { DTLSErrorCodes } from "../lsp/methods/textDocument/diagnostic.ts";

import { Logger } from "#logger";
import { TokenMap } from "#tokens";
import { DTLSContext } from "#lsp";

const f = await Deno.open(
  new URL("./tree-sitter/tree-sitter-css.wasm", import.meta.url),
);

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
  if (!tree) {
    return [];
  }
  const query = new Query(tree.language, LightDarkValuesQuery);
  const captures = query.captures(tree.rootNode);
  const lightNode = captures.find((cap) => cap.name === "lightValue");
  const darkNode = captures.find((cap) => cap.name === "darkValue");
  return [lightNode?.node.text, darkNode?.node.text];
}

export function tsRangeToLspRange(node: TsRange | Node): LSP.Range {
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

export function lspRangeToTsRange(range: LSP.Range): TsRange {
  return {
    startPosition: {
      row: range.start.line,
      column: range.start.character,
    },
    endPosition: {
      row: range.end.line,
      column: range.end.character,
    },
  };
}

export function lspPosToTsPos(pos: LSP.Position): Point {
  return {
    row: pos.line,
    column: pos.character,
  };
}

export function tsNodeIsInLspRange(
  node: TSNodePosition,
  range: LSP.Range,
): boolean {
  const inRows = node.startPosition.row >= range.start.line &&
    node.endPosition.row <= range.end.line;
  const inCols = node.startPosition.column >= range.start.character &&
    node.endPosition.column <= range.end.character;
  return (inRows && inCols);
}

export function lspRangeIsInTsNode(
  node: TSNodePosition,
  range: LSP.Range,
): boolean {
  const inRows = node.startPosition.row >= range.start.line &&
    node.endPosition.row <= range.end.line;
  const inCols = node.startPosition.column <= range.start.character &&
    node.endPosition.column >= range.end.character;
  return (inRows && inCols);
}

export function captureIsTokenName(cap: QueryCapture, { tokens }: DTLSContext) {
  return cap.name === "tokenName" &&
    tokens.has(cap.node.text.replace(/^--/, ""));
}

export function captureIsTokenCall(cap: QueryCapture, { tokens }: DTLSContext) {
  return cap.name === "call" && !!cap.node.children
    .find((child) => child?.type === "arguments")
    ?.children
    .some((child) =>
      child?.type === "plain_value" &&
      tokens.has(child?.text.replace(/^--/, ""))
    );
}

class ENODOCError extends Error {
  constructor(public uri: LSP.DocumentUri) {
    super(`ENOENT: no CssDocument found for ${uri}`);
  }
}

export class CssDocument extends FullTextDocument {
  #tree: Tree | null;
  #tokens: TokenMap;

  diagnostics: LSP.Diagnostic[];

  constructor(
    uri: string,
    languageId: string,
    version: number,
    text: string,
    tokens: TokenMap,
  ) {
    super(uri, languageId, version, text);
    this.#tokens = tokens;
    this.#tree = parser.parse(text);
    this.diagnostics = this.#computeDiagnostics();
  }

  override update(
    changes: LSP.TextDocumentContentChangeEvent[],
    version: number,
  ) {
    const old = this.getText();
    super.update(changes, version);
    const newText = this.getText();
    const newRows = newText.split("\n");
    if (!this.#tree) {
      return;
    }
    const oldEndPosition = this.#tree.rootNode.endPosition;
    this.#tree.edit({
      startIndex: 0,
      oldEndIndex: old.length,
      newEndIndex: newText.length,
      startPosition: { row: 0, column: 0 },
      oldEndPosition,
      newEndPosition: {
        row: newRows.length - 1,
        column: newRows[newRows.length - 1].length - 1,
      },
    });
    this.#tree = parser.parse(newText, this.#tree);
    this.diagnostics = this.#computeDiagnostics();
  }

  /**
   * Queries the document for a specific query.
   *
   * @param query - The query to run.
   * @param options - The options to pass to the query.
   */
  query(query: string, options?: TSNodePosition) {
    if (!this.#tree) {
      return [];
    }
    const q = new Query(this.#tree.language, query);
    return q.captures(this.#tree.rootNode, { matchLimit: 65536, ...options });
  }

  /**
   * Gets the node at the specified position in the document.
   *
   * @param position - The position to check.
   */
  getNodeAtPosition(position: LSP.Position): null | Node {
    return this.#tree?.rootNode.descendantForPosition({
      row: position.line,
      column: position.character,
    }) ?? null;
  }

  /**
   * Checks if the given position is within a specific node type in the document.
   *
   * @param position - The position to check.
   * @param type - The type of node to check against.
   * @return whether the position is within the specified node type.
   */
  positionIsInNodeType(position: LSP.Position, type: string): boolean {
    let node = this.getNodeAtPosition(position);
    while (node && node.type !== "stylesheet") {
      if (node.type === type) {
        return true;
      } else {
        node = node.parent;
      }
    }
    return false;
  }

  #computeDiagnostics() {
    const captures = this.query(VarCallWithFallback);
    const tokenNameCaps = captures.filter((x) => x.name === "tokenName");
    const fallbackCaps = captures.filter((x) => x.name === "fallback");
    return zip(tokenNameCaps, fallbackCaps).flatMap(
      ([tokenNameCap, fallbackCap]) => {
        if (this.#tokens.has(tokenNameCap.node.text)) {
          const tokenName = tokenNameCap.node.text;
          const fallback = fallbackCap.node.text;
          const token = this.#tokens.get(tokenName)!;
          const valid = fallback === token.$value;
          if (!valid) {
            return [{
              range: tsRangeToLspRange(fallbackCap.node),
              severity: LSP.DiagnosticSeverity.Error,
              message:
                `Token fallback does not match expected value: ${token.$value}`,
              code: DTLSErrorCodes.incorrectFallback,
              data: {
                tokenName,
              },
            }];
          }
        }
        return [];
      },
    );
  }
}

export class Documents {
  #map = new Map<LSP.DocumentUri, CssDocument>();

  get handlers() {
    return {
      "textDocument/didOpen": (
        params: LSP.DidOpenTextDocumentParams,
        context: DTLSContext,
      ) => this.onDidOpen(params, context.tokens),
      "textDocument/didChange": (params: LSP.DidChangeTextDocumentParams) =>
        this.onDidChange(params),
      "textDocument/didClose": (params: LSP.DidCloseTextDocumentParams) =>
        this.onDidClose(params),
    } as const;
  }

  protected get allDocuments() {
    return [...this.#map.values()];
  }

  onDidOpen(params: LSP.DidOpenTextDocumentParams, tokens: TokenMap) {
    const { uri, languageId, version, text } = params.textDocument;
    const doc = new CssDocument(uri, languageId, version, text, tokens);
    Logger.debug`📖 Opened ${uri}`;
    this.#map.set(params.textDocument.uri, doc);
  }

  onDidChange(params: LSP.DidChangeTextDocumentParams) {
    const { uri, version } = params.textDocument;
    const doc = this.get(uri);
    doc.update(params.contentChanges, version);
  }

  onDidClose(params: LSP.DidCloseTextDocumentParams) {
    this.#map.delete(params.textDocument.uri);
  }

  get(uri: LSP.DocumentUri) {
    const doc = this.#map.get(uri);
    if (!doc) {
      throw new ENODOCError(uri);
    }
    return doc;
  }

  getVersion(uri: LSP.DocumentUri) {
    const doc = this.get(uri);
    return doc.version;
  }

  getText(uri: LSP.DocumentUri) {
    return this.get(uri).getText();
  }

  getDiagnostics(uri: LSP.DocumentUri) {
    const doc = this.get(uri);
    return doc.diagnostics;
  }

  getNodeAtPosition(uri: LSP.DocumentUri, position: LSP.Position) {
    return this.get(uri).getNodeAtPosition(position);
  }

  queryVarCalls(uri: LSP.DocumentUri) {
    return this.get(uri).query(VarCall);
  }

  queryVarCallsWithFallback(uri: LSP.DocumentUri) {
    return this.get(uri).query(VarCallWithFallback);
  }
}

export const documents = new Documents();
