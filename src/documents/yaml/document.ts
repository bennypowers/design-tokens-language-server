import * as LSP from "vscode-languageserver-protocol";

import {
  adjustPosition,
  DTLSTextDocument,
  TOKEN_REFERENCE_REGEXP,
} from "#document";

import { DTLSContext, DTLSErrorCodes } from "#lsp/lsp.ts";

import * as YAML from "yaml";

import { usesReferences } from "style-dictionary/utils";

import { cssColorToLspColor } from "#color";
import { getLightDarkValues } from "#css";
import { Logger } from "#logger";
import { DTLSToken } from "#tokens";

import {
  DTLSSemanticTokenIntermediate,
  DTLSTokenTypes,
} from "#methods/textDocument/semanticTokens.ts";

type YAMLPath = readonly (
  | YAML.Node
  | YAML.Document<YAML.Node, true>
  | YAML.Pair<unknown, unknown>
)[];

export class YamlDocument extends DTLSTextDocument {
  language = "yaml" as const;

  #root!: YAML.Node;
  #document!: YAML.Document.Parsed;
  #lineCounter!: YAML.LineCounter;

  #context!: DTLSContext;

  static create(context: DTLSContext, uri: string, text: string, version = 0) {
    const doc = new YamlDocument(uri, version, text);
    doc.#context = context;
    return doc;
  }

  private constructor(uri: string, version: number, text: string) {
    super(uri, "yaml", version, text);
    this.#parse();
  }

  override update(
    changes: LSP.TextDocumentContentChangeEvent[],
    version: number,
  ) {
    super.update(changes, version);
    this.#parse();
  }

  #parse() {
    const lineCounter = new YAML.LineCounter();
    this.#document = YAML.parseDocument(this.getText(), {
      lineCounter,
    });
    this.#lineCounter = lineCounter;
    const root = this.#document.contents;
    if (!root) {
      throw new Error("Failed to parse YAML");
    }
    this.#root = root;
  }

  #getRangeForNode(node: YAML.Node | null) {
    if (!node?.range) return null;
    const [startOffset, valueEndOffset] = node.range;
    const startPos = this.#lineCounter.linePos(startOffset);
    const endPos = this.#lineCounter.linePos(valueEndOffset);
    return {
      start: { line: startPos.line - 1, character: startPos.col - 1 },
      end: { line: endPos.line - 1, character: endPos.col - 1 },
    };
  }

  /**
   * cheap hack to avoid marking up non-color values
   * a more comprehensive solution would be to associate each json document
   * with a token spec, and get the token object *with prefix* from the context
   * based on the *non-prefixed* json path, then check the type from the token object
   */
  #getDTCGTypeForNode(node: YAML.Node, path?: YAMLPath) {
    if (!path) {
      YAML.visit(this.#root, {
        Node(_, n, p) {
          if (n === node) {
            path = p;
            return YAML.visit.BREAK;
          }
        },
      });
    }
    if (!path) throw new Error("Could not find node");
    const pathMut = [...path];
    let parent: unknown = node;
    let typeNode;
    let type: string | undefined;
    while (type !== "color" && (parent = pathMut.pop())) {
      if (YAML.isMap(parent)) {
        typeNode = parent.get("$type", true);
        type = YAML.isScalar<string>(typeNode) ? typeNode.value : undefined;
      }
    }
    return type;
  }

  #getNodeAtPath(
    path: (string | number)[],
  ): YAML.Scalar | YAML.YAMLMap | YAML.YAMLSeq | null {
    const node = this.#document.getIn(path, true);
    if (YAML.isAlias(node)) {
      return node.resolve(this.#document) ?? null;
    } else if (YAML.isNode(node) && !YAML.isAlias(node)) {
      return node;
    } else if (node) {
      Logger.warn`Unknown ${node} ${typeof node}`;
    }
    return null;
  }

  #getOffsetAtPosition(position: LSP.Position): number {
    return this.getText()
      .split("\n")
      .reduce((acc, row, i) => {
        if (i === position.line) {
          return acc + position.character;
        } else if (i < position.line) return acc + row.length + 1; // add 1 for the '\n'
        else return acc;
      }, 0);
  }

  #getNodeAtPosition(
    position: LSP.Position,
    offset: Partial<LSP.Position> = {},
  ): YAML.Scalar | YAML.YAMLMap | YAML.YAMLSeq | null {
    let found: YAML.Node | null = null;
    const adjustedPosition = adjustPosition(position, offset);
    // convert the line/col position to a numeric offset
    const offsetOfPosition = this.#getOffsetAtPosition(adjustedPosition);

    function isInRangeOfPosition(node: YAML.Node) {
      const [start, end] = node?.range ?? [Infinity, -Infinity];
      return start <= offsetOfPosition && end >= offsetOfPosition;
    }

    YAML.visit(this.#root, {
      Node: (_, unresolved) => {
        let node: YAML.Node | undefined = unresolved;
        if (YAML.isAlias(unresolved)) {
          node = unresolved.resolve(this.#document);
        }
        if (YAML.isScalar(node) && isInRangeOfPosition(node)) {
          found = node;
        } else if (YAML.isSeq<YAML.Node>(node)) {
          found = node.items.find(isInRangeOfPosition) ?? null;
        }
        if (found) {
          return YAML.visit.BREAK;
        }
      },
    });

    return found;
  }

  #getNodeForTokenName(tokenName: string, prefix?: string): YAML.Node | null {
    const tokenPath = tokenName
      .replace(/^--/, "")
      .split("-")
      .filter((x) => !!x)
      .filter((x) => (prefix ? x !== prefix : true));
    const node = this.#getNodeAtPath(tokenPath);
    return node ?? null;
  }

  getTokenForPath(path: (string | number)[]): DTLSToken | null {
    const node = this.#getNodeAtPath(path);
    const valueNode = this.#getNodeAtPath([...path, "$value"]);
    if (!node || !valueNode) {
      return null;
    }
    const getValues = (node?: YAML.Node): unknown[] => {
      if (YAML.isScalar(node)) return [node?.value];
      else if (YAML.isAlias(node)) {
        return getValues(node.resolve(this.#document));
      } else if (YAML.isMap(node)) {
        return node.items.flatMap((pair) => getValues(pair.value as YAML.Node));
      } else if (YAML.isSeq(node)) {
        return node.items.flatMap((item) => getValues(item as YAML.Node));
      } else return [];
    };

    let $value = getValues(valueNode).join(""); // XXX: this join may come back to bite me
    if ($value) {
      if (usesReferences($value)) {
        const resolved = this.#context.tokens.resolveValue($value)?.toString();
        if (resolved) {
          $value = resolved;
        }
      }
      const prefix = this.#context.workspaces.getPrefixForUri(this.uri);
      return this.#context.tokens.get(
        [prefix, ...path].filter((x) => !!x).join("-"),
      ) ?? null;
    }
    return null;
  }

  #localReferenceToTokenName(reference: string) {
    const prefix = this.#context.workspaces.getPrefixForUri(this.uri);
    const path = Array.isArray(reference) ? reference : reference.split(".");
    return [prefix, ...path].filter((x) => !!x).join("-");
  }

  getRangeForTokenName(tokenName: string, prefix?: string): LSP.Range | null {
    return this.#getRangeForNode(this.#getNodeForTokenName(tokenName, prefix));
  }

  getRangeForPath(path: (string | number)[]): LSP.Range | null {
    return this.#getRangeForNode(this.#getNodeAtPath(path));
  }

  getTokenReferenceAtPosition(
    position: LSP.Position,
    offset: Partial<LSP.Position> = {},
  ) {
    const adjusted = adjustPosition(position, offset);
    const stringNode = this.#getNodeAtPosition(adjusted, offset);
    if (!YAML.isScalar(stringNode)) {
      return null;
    }
    if (typeof stringNode.value === "string") {
      const valueRange = this.#getRangeForNode(stringNode);

      if (!valueRange) return null;

      const valueLines = this.getText()
        .split("\n")
        .slice(valueRange.start.line, valueRange.end.line + 1);

      // get the match that contains the current position
      for (const currentLine of valueLines) {
        const matches = currentLine.match(TOKEN_REFERENCE_REGEXP); // ['{color.blue._}', '{color.blue.dark}']
        for (const reference of matches ?? []) {
          const matchOffsetInLine = currentLine.indexOf(reference);
          if (
            adjusted.character >= matchOffsetInLine &&
            adjusted.character <= matchOffsetInLine + reference.length
          ) {
            const stringRange = this.#getRangeForNode(stringNode)!;
            const refUnpacked = reference.replace(TOKEN_REFERENCE_REGEXP, "$1");
            const line = stringRange.start.line;
            const character = currentLine.indexOf(refUnpacked);
            const name = `--${this.#localReferenceToTokenName(refUnpacked)}`;
            if (this.#context.tokens.has(name)) {
              return {
                name,
                range: {
                  start: { line, character },
                  end: { line, character: character + reference.length - 2 },
                },
              };
            }
          }
        }
      }
    }
    return null;
  }

  public getSemanticTokensFull() {
    const tokens: DTLSSemanticTokenIntermediate[] = [];
    const addToken = (value: YAML.Scalar) => {
      if (typeof value.value === "string" && usesReferences(value.value)) {
        const valueNodeRange = this.#getRangeForNode(value);
        if (!valueNodeRange) return;
        const valueLines = value.value.split("\n");
        tokens.push(...valueLines.flatMap((currentLine, i) => {
          const line = valueNodeRange.start.line + i;
          const matches = currentLine.matchAll(TOKEN_REFERENCE_REGEXP);
          if (!matches) return [];
          return matches.flatMap((match) => {
            const { reference } = match.groups!;
            const name = this.#localReferenceToTokenName(reference);
            if (!this.#context.tokens.has(name)) return [];
            const [start] = match.indices!.groups!.reference;
            let lastStartChar = 1 +
              (i === 0
                ? valueNodeRange.start.character + start
                : currentLine.indexOf(reference));
            return reference.split(".").map((token, k) => {
              const startChar = lastStartChar;
              const tokenType = k === 0
                ? DTLSTokenTypes.at(0)!
                : DTLSTokenTypes.at(1)!;
              const { length } = token;
              lastStartChar += token.length + 1;
              return {
                token,
                line,
                startChar,
                length,
                tokenType,
                tokenModifiers: 0,
              };
            });
          }).toArray();
        }));
      }
    };
    YAML.visit(this.#root, {
      Pair: (_, node) => {
        const { key, value } = node;
        if (YAML.isScalar(key) && key.value === "$value") {
          if (YAML.isScalar(value)) {
            addToken(value);
          } else if (YAML.isSeq(value)) {
            for (const item of value.items) {
              if (YAML.isScalar(item)) {
                addToken(item);
              }
            }
          }
        }
      },
    });
    return tokens.sort((a, b) => {
      if (a.line === b.line) return a.startChar - b.startChar;
      else return a.line - b.line;
    });
  }

  public getColors(context: DTLSContext): LSP.ColorInformation[] {
    const colors: LSP.ColorInformation[] = [];
    const addColor = (
      value: YAML.Scalar,
      path: readonly (
        | YAML.Node
        | YAML.Pair<unknown, unknown>
        | YAML.Document<YAML.Node, true>
      )[],
    ) => {
      if (typeof value.value === "string") {
        const content = value.value;
        const type = this.#getDTCGTypeForNode(value, path);
        if (type === "color" && value.range) {
          const range = this.#getRangeForNode(value);
          if (!range) return;
          if (range && usesReferences(content)) {
            const references = content.match(/{[^}]*}/g);
            for (const reference of references ?? []) {
              const resolved = context.tokens.resolveValue(reference);
              if (resolved) {
                const line = range.start.line;
                const character = range.start.character +
                  content.indexOf(reference) + 1;
                const color = cssColorToLspColor(resolved.toString());
                if (color) {
                  colors.push({
                    color,
                    range: {
                      start: { line, character },
                      end: {
                        line,
                        character: character + reference.length - 2,
                      },
                    },
                  });
                }
              }
            }
          } else if (content.startsWith("light-dark(")) {
            for (const match of getLightDarkValues(content)) {
              const color = cssColorToLspColor(match);
              if (color) {
                colors.push({ range, color });
              }
            }
          } else {
            const resolved = context.tokens.resolveValue(content);
            const match = resolved?.toString() ?? content;
            const color = cssColorToLspColor(match);
            if (color) {
              colors.push({ range, color });
            }
          }
        }
      }
    };
    YAML.visit(this.#root, {
      Pair: (_key, node, path) => {
        if (YAML.isScalar(node.key) && node.key.value === "$value") {
          if (YAML.isScalar(node.value)) {
            addColor(node.value, path);
          } else if (YAML.isSeq(node.value)) {
            for (const item of node.value.items) {
              if (YAML.isScalar(item)) {
                addColor(item, path);
              }
            }
          }
        }
      },
    });
    return colors;
  }

  public getCompletions(
    context: DTLSContext,
    params: LSP.CompletionParams,
  ): LSP.CompletionList | null {
    const node = this.#getNodeAtPosition(params.position);
    if (YAML.isScalar(node) && typeof node.value === "string") {
      const prefix = node.value.replace(/}$/, "");
      const range = this.#getRangeForNode(node);
      if (range) {
        const items = context.tokens
          .values()
          .flatMap((token) => {
            const ext = token.$extensions.designTokensLanguageServer;
            if (!ext.reference.startsWith(prefix)) return [];
            const label = `'${ext.reference}'`;
            const tokenName = ext.name;
            return [{ label, data: { tokenName } }];
          })
          .toArray();

        return {
          items,
          isIncomplete: items.length === 0 ||
            items.length < context.tokens.size,
          itemDefaults: {
            insertTextFormat: LSP.InsertTextFormat.Snippet,
            insertTextMode: LSP.InsertTextMode.asIs,
            editRange: range,
          },
        };
      }
    }
    return null;
  }

  public getDiagnostics(context: DTLSContext) {
    if (!context.tokens) {
      throw new Error("No tokens found in context");
    }
    const diagnostics: LSP.Diagnostic[] = [];
    // TODO: start with Pair, like in Color, and generalize that code
    YAML.visit(this.#root, {
      Scalar: (_, node) => {
        const range = this.#getRangeForNode(node);
        const content = node.value;
        if (!range || typeof content !== "string") return;
        const errors: string[] = [];
        if (usesReferences(content)) {
          const matches = content.match(TOKEN_REFERENCE_REGEXP);
          for (const name of matches ?? []) {
            const resolved = context.tokens.resolveValue(name);
            if (!resolved) {
              errors.push(name);
            }
          }
        }
        diagnostics.push(
          ...errors.map((name) => ({
            range,
            severity: LSP.DiagnosticSeverity.Error,
            message: `Token reference does not exist: ${name}`,
            code: DTLSErrorCodes.unknownReference,
          })),
        );
      },
    });
    return diagnostics;
  }

  public override getDocumentSymbols(
    context: DTLSContext,
  ): LSP.DocumentSymbol[] {
    // TODO: return tokens
    return [];
  }
}
