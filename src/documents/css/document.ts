import { Node, Point, Query, QueryCapture, Tree } from "web-tree-sitter";

import { Logger } from "#logger";
import { DTLSContext, DTLSErrorCodes } from "#lsp";
import { DTLSTextDocument } from "#document";

import * as LSP from "vscode-languageserver-protocol";

import * as Queries from "./tree-sitter/queries.ts";

import { parser } from "./tree-sitter/parser.ts";

type TsRange = Pick<Node, "startPosition" | "endPosition">;

export interface TSNodePosition {
  endPosition: { row: number; column: number };
  startPosition: { row: number; column: number };
}

export function getLightDarkValues(value: string) {
  const tree = parser.parse(`a{b:${value}}`)!;
  const query = new Query(tree.language, Queries.LightDarkValuesQuery);
  const captures = query.captures(tree.rootNode);
  const lightNode = captures.find((cap) => cap.name === "lightValue");
  const darkNode = captures.find((cap) => cap.name === "darkValue");
  return [lightNode?.node.text, darkNode?.node.text].filter((x) =>
    x !== undefined
  );
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

function tsNodesToLspRangeInclusive(...nodes: TsRange[]): LSP.Range {
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

function lspRangeToTsRange(range: LSP.Range): TsRange {
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

function lspPosToTsPos(pos: LSP.Position): Point {
  return {
    row: pos.line,
    column: pos.character,
  };
}

function offsetPosition(
  position: LSP.Position,
  offset: Partial<LSP.Position>,
): LSP.Position {
  return {
    line: position.line + (offset.line ?? 0),
    character: position.character + (offset.character ?? 0),
  };
}

export class CssDocument extends DTLSTextDocument {
  static create(
    context: DTLSContext,
    uri: string,
    text: string,
    version = 0,
  ) {
    const doc = new CssDocument(uri, version, text);
    doc.#tree = parser.parse(text);
    doc.diagnostics = doc.computeDiagnostics(context);
    return doc;
  }

  language = "css" as const;

  #tree!: Tree | null;

  static queries = Queries;

  private constructor(
    uri: string,
    version: number,
    text: string,
  ) {
    super(uri, "css", version, text);
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
  getNodeAtPosition(
    position: LSP.Position,
    offset?: Partial<LSP.Position>,
  ): null | Node {
    const pos = !offset ? position : offsetPosition(position, offset);
    return this.#tree?.rootNode.descendantForPosition(lspPosToTsPos(pos)) ??
      null;
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

  override computeDiagnostics(context: DTLSContext) {
    const captures = this.query(Queries.VarCallWithFallback);

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
