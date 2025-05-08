import { Node, Point, Query, Tree } from "web-tree-sitter";

import { DTLSContext, DTLSErrorCodes } from "#lsp/lsp.ts";
import { DTLSTextDocument, TokenReference } from "#document";

import * as LSP from "vscode-languageserver-protocol";

import * as Queries from "./tree-sitter/queries.ts";

import { parser } from "./tree-sitter/parser.ts";
import { Token } from "style-dictionary";

import { cssColorToLspColor } from "#color";
import { DTLSToken } from "#tokens";

/**
 * Regular expression to match hex color values.
 */
const HEX_RE = /#(?<hex>.{3}|.{4}|.{6}|.{8})\b/g;

interface CompletionArgs {
  node: Node;
  name: string;
  range: LSP.Range;
  tokens: DTLSContext["tokens"];
}

type TsRange = Pick<Node, "startPosition" | "endPosition">;

export interface TokenVarCall {
  range: LSP.Range;
  token: {
    name: string;
    range: LSP.Range;
    token: Token;
  };
  fallback?: {
    value: string;
    valid: boolean;
    range: LSP.Range;
  };
}

export interface TSNodePosition {
  endPosition: { row: number; column: number };
  startPosition: { row: number; column: number };
}

export function getLightDarkValues(value: string) {
  const tree = parser.parse(`a{b:${value}}`)!;
  const query = new Query(tree.language, Queries.VarCallWithLightDarkFallback);
  const captures = query.captures(tree.rootNode);
  const lightNode = captures.find((cap) => cap.name === "lightValue");
  const darkNode = captures.find((cap) => cap.name === "darkValue");
  return [lightNode?.node.text, darkNode?.node.text].filter((x) =>
    x !== undefined
  );
}

export function getVarCallArguments(value: string) {
  const tree = parser.parse(`a{b:${value}}`)!;
  const query = new Query(tree.language, Queries.VarCallWithOrWithoutFallback);
  const captures = query.captures(tree.rootNode);
  const tokenNameNode = captures.find((cap) => cap.name === "tokenName");
  const fallback = value.replace(
    new RegExp(`^var\\(${tokenNameNode?.node.text}(, *)`),
    "",
  )
    .replace(/\)$/, "")
    .trim();
  return {
    variable: tokenNameNode?.node.text,
    fallback,
  };
}

function tsRangeToLspRange(node: TsRange | Node): LSP.Range {
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
      return `${name}: $0`;
    default: {
      const token = tokens.get(name)!;
      const value = Array.isArray(token.$value)
        ? token.$value.join(", ")
        : token.$value;
      return `var(${name}\${1|\\, ${escapeCommas(value)},|})$0`;
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
 * Given that the match can be a hex color, a css color name, or a var call,
 * and that if it's a var call, it can be to a known token, or to an unknown
 * custom property, we need to extract the color value from the match.
 * We can't return the var call as-is, because tinycolor can't parse it.
 * So we need to return the fallback value of the var call, which itself could be a var call or
 * any valid css color value.
 *
 * We also need to handle the case where the var call is a known token, in which case we can just
 * return the value of the token.
 */
function extractColor(match: string, context: DTLSContext): string {
  if (match.startsWith("var(")) {
    const { variable, fallback } = getVarCallArguments(match);
    if (context.tokens.has(variable)) {
      return extractColor(context.tokens.get(variable)!.$value, context);
    } else if (fallback) {
      return extractColor(fallback, context);
    }
  }
  return match;
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
    doc.#context = context;
    doc.#varCalls = doc.#computeVarCalls(context);
    doc.#diagnostics = doc.#computeDiagnostics();
    return doc;
  }

  #tree!: Tree | null;
  #context!: DTLSContext;
  #varCalls: TokenVarCall[] = [];
  #diagnostics: LSP.Diagnostic[] = [];

  language = "css" as const;

  /**
   * TODO: having this on CSSDocument and not JsonDocument is a code smell
   * This is ultimately meant to get diagnostics and code actions
   * eventually, we'll compute all of those upfront and store them here,
   * then use code action resolve for the details
   */
  get varCalls() {
    return this.#varCalls;
  }

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
    this.#varCalls = this.#computeVarCalls(this.#context);
    this.#diagnostics = this.#computeDiagnostics();
  }

  #computeVarCalls(context: DTLSContext): TokenVarCall[] {
    const captures = this.query(Queries.VarCallWithOrWithoutFallback);

    const callNodes = new Map<
      number,
      {
        range: LSP.Range;
        tokenNameNode: Node;
        fallback: string;
        fallbacks: Node[];
      }
    >();

    for (const cap of captures) {
      if (cap.name === "VarCallWithOrWithoutFallback") {
        const { fallback } = getVarCallArguments(cap.node.text);
        callNodes.set(cap.node.id, {
          tokenNameNode: {} as Node,
          range: tsRangeToLspRange(cap.node),
          fallback,
          fallbacks: [],
        });
      }
    }

    for (const cap of captures) {
      let node = cap.node;
      let callNode = node.parent;
      while (callNode?.type !== "call_expression" && node.parent) {
        callNode = node.parent;
        node = node.parent;
      }
      if (callNode?.type === "call_expression") {
        if (cap.name === "tokenName") {
          callNodes.get(callNode.id)!.tokenNameNode = cap.node;
        } else if (cap.name === "fallback") {
          callNodes.get(callNode.id)!.fallbacks.push(cap.node);
        }
      }
    }

    return callNodes.values().flatMap(
      ({ range, tokenNameNode, fallbacks, fallback }) => {
        const _token = context.tokens.get(tokenNameNode.text)!;
        const token = {
          name: tokenNameNode.text,
          range: tsRangeToLspRange(tokenNameNode),
          token: _token,
        };
        if (context.tokens.has(token.name)) {
          const { $value } = token.token;
          if (fallbacks.length) {
            const value = $value.toString();
            // TODO: compute semantic equivalence using CSS OM
            const valid = _token.$type === "color"
              ? fallback.toLowerCase() === value.toLowerCase()
              : fallback === value;
            return [{
              range,
              token,
              fallback: {
                value: fallback,
                valid,
                range: tsNodesToLspRangeInclusive(...fallbacks),
              },
            }];
          } else {
            return [{ range, token }];
          }
        }
        return [];
      },
    ).toArray();
  }

  #computeDiagnostics() {
    return this.varCalls.flatMap((call) => {
      if (!call.fallback || call.fallback.valid) return [];
      else {
        const actual = call.fallback.value;
        const expected = call.token.token.$value;
        const tokenName = call.token.name;
        return [{
          range: call.fallback.range,
          severity: LSP.DiagnosticSeverity.Error,
          message: `Token fallback does not match expected value: ${expected}`,
          code: DTLSErrorCodes.incorrectFallback,
          data: {
            tokenName,
            actual,
            expected,
          },
        }];
      }
    });
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

  getTokenReferenceAtPosition(
    position: LSP.Position,
    offset?: Partial<LSP.Position>,
  ): TokenReference | null {
    const node = this.getNodeAtPosition(position, offset);
    if (node) {
      const name = `--${node.text}`.replace("----", "--");
      if (this.#context.tokens.has(name)) {
        return { name, range: tsRangeToLspRange(node) };
      }
    }
    return null;
  }

  getRangeForPath(_: string[]): LSP.Range | null {
    throw new Error("Cannot perform path operations in CSS documents");
  }

  getTokenForPath(_: string[]): DTLSToken | null {
    throw new Error("Cannot perform path operations in CSS documents");
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

  public getColors(context: DTLSContext): LSP.ColorInformation[] {
    this.#context = context;
    return this.#varCalls.flatMap((call) => {
      const token = call.token.token;

      if (!token || token.$type !== "color") {
        return [];
      }
      const colors = [];
      const hexMatches = `${token.$value}`.match(HEX_RE);
      const [light, dark] = getLightDarkValues(token.$value);
      if (light && dark) {
        colors.push(light, dark);
      } else if (hexMatches) {
        colors.push(...hexMatches);
      } else {
        colors.push(token.$value);
      }
      return colors.flatMap((match) => {
        const colorMatch = extractColor(match, this.#context);
        const color = cssColorToLspColor(colorMatch);
        if (!color) {
          return [];
        } else {
          return [{
            color,
            range: call.token.range,
          }];
        }
      });
    });
  }

  public getDiagnostics(context: DTLSContext) {
    this.#context = context;
    return this.#diagnostics ?? this.#computeDiagnostics();
  }

  public getCompletions(
    context: DTLSContext,
    params: LSP.CompletionParams,
  ): LSP.CompletionList | null {
    // The { character: -2 } offset adjusts the position slightly backward to ensure
    // that the correct node is retrieved, especially when the cursor is at the end
    // of a token. This helps avoid off-by-one errors in node selection.
    const node = this.getNodeAtPosition(params.position, { character: -2 });

    if (
      !node || node.type !== "identifier" &&
        !this.positionIsInNodeType(params.position, "block")
    ) {
      return null;
    }

    const range = tsRangeToLspRange(node);
    const items = context.tokens
      .keys()
      .filter(function (name: string): boolean {
        return !!node.text && !!name && name
          .replaceAll("-", "")
          .startsWith(node.text.replaceAll("-", ""));
      })
      .map((name) =>
        ({
          label: name,
          data: { tokenName: name },
          kind: LSP.CompletionItemKind.Snippet,
          ...getEditOrEntry({ node, name, range, tokens: context.tokens }),
        }) satisfies LSP.CompletionItem
      ).toArray();

    return {
      items,
      isIncomplete: items.length === 0 || items.length < context.tokens.size,
      itemDefaults: {
        insertTextFormat: LSP.InsertTextFormat.Snippet,
        insertTextMode: LSP.InsertTextMode.asIs,
        editRange: range,
      },
    };
  }
}
