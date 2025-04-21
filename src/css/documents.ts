import { Language, Parser, Query } from "web-tree-sitter";
import type { Node, Point, QueryCapture, Tree } from "web-tree-sitter";

import { readAll } from "jsr:@std/io/read-all";

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

export function tsNodesToLspRangeInclusive(...nodes: TsRange[]): LSP.Range {
  const [startNode] = nodes;
  const endNode = nodes.pop()!;
  const start = {
    line: startNode.startPosition.row,
    character: startNode.startPosition.column,
  };
  const end = {
    line: endNode.endPosition.row,
    character: endNode.endPosition.column,
  };
  return { start, end };
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

  diagnostics: LSP.Diagnostic[];

  constructor(
    uri: string,
    languageId: string,
    version: number,
    text: string,
    context: DTLSContext,
  ) {
    super(uri, languageId, version, text);
    this.#tree = parser.parse(text);
    this.diagnostics = this.computeDiagnostics(context);
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

  computeDiagnostics(context: DTLSContext) {
    const captures = this.query(VarCallWithFallback);

    const callNodes = new Map<
      number,
      { tokenName: string; fallbacks: Node[] }
    >();

    for (const cap of captures) {
      if (cap.name === "VarCallWithFallback") {
        callNodes.set(cap.node.id, { tokenName: "", fallbacks: [] });
      }
    }

    for (const cap of captures) {
      const callNode = cap.node.parent?.parent;
      if (callNode?.type === "call_expression") {
        try {
          if (cap.name === "tokenName") {
            callNodes.get(callNode.id)!.tokenName = cap.node.text;
          } else if (cap.name === "fallback") {
            callNodes.get(callNode.id)!.fallbacks.push(cap.node);
          }
        } catch (e) {
          Logger.error`Error while computing diagnostics: ${e}`;
        }
      }
    }

    return callNodes.values().flatMap(({ tokenName, fallbacks }) => {
      if (context.tokens.has(tokenName)) {
        const token = context.tokens.get(tokenName)!;
        const joiner = token.$type === "fontFamily" ? ", " : " ";
        const fallback = fallbacks.map((x) => x.text).join(joiner);
        const valid = typeof token.$value === "number"
          ? (parseFloat(fallback) === token.$value)
          : (fallback === token.$value);
        if (!valid) {
          return [{
            range: tsNodesToLspRangeInclusive(...fallbacks),
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
    }).toArray();
  }
}

export class Documents {
  #map = new Map<LSP.DocumentUri, CssDocument>();

  get handlers() {
    return {
      "textDocument/didOpen": (
        params: LSP.DidOpenTextDocumentParams,
        context: DTLSContext,
      ) => this.onDidOpen(params, context),
      "textDocument/didChange": (
        params: LSP.DidChangeTextDocumentParams,
        context: DTLSContext,
      ) => this.onDidChange(params, context),
      "textDocument/didClose": (
        params: LSP.DidCloseTextDocumentParams,
        context: DTLSContext,
      ) => this.onDidClose(params, context),
    } as const;
  }

  protected get allDocuments() {
    return [...this.#map.values()];
  }

  onDidOpen(params: LSP.DidOpenTextDocumentParams, context: DTLSContext) {
    const { uri, languageId, version, text } = params.textDocument;
    const doc = new CssDocument(uri, languageId, version, text, context);
    Logger.debug`📖 Opened ${uri}`;
    this.#map.set(params.textDocument.uri, doc);
  }

  onDidChange(params: LSP.DidChangeTextDocumentParams, context: DTLSContext) {
    const { uri, version } = params.textDocument;
    const doc = this.get(uri);
    doc.update(params.contentChanges, version);
    doc.diagnostics = doc.computeDiagnostics(context);
  }

  onDidClose(params: LSP.DidCloseTextDocumentParams, _: DTLSContext) {
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
