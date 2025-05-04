import * as LSP from "vscode-languageserver-protocol";

import { adjustPosition, DTLSTextDocument } from "#document";

import { DTLSContext, DTLSErrorCodes } from "#lsp/lsp.ts";

import * as YAML from "yaml";

import { cssColorToLspColor } from "#color";
import { usesReferences } from "style-dictionary/utils";
import { getLightDarkValues } from "#css";
import { Logger } from "#logger";
import { DTLSToken } from "#tokens";

const REF_RE = /{([^}]+)}/g;

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
    this.#root = this.#parse();
  }

  override update(
    changes: LSP.TextDocumentContentChangeEvent[],
    version: number,
  ) {
    super.update(changes, version);
    this.#root = this.#parse();
  }

  #parse(): YAML.Node {
    const lineCounter = new YAML.LineCounter();
    this.#document = YAML.parseDocument(this.getText(), {
      lineCounter,
    });
    this.#lineCounter = lineCounter;
    const root = this.#document.contents;
    if (!root) {
      throw new Error("Failed to parse YAML");
    }
    return root;
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

  #getNodeAtPosition(
    position: LSP.Position,
    offset: Partial<LSP.Position> = {},
  ): YAML.Scalar | YAML.YAMLMap | YAML.YAMLSeq | null {
    let found: YAML.Node | null = null;
    const adjustedPosition = adjustPosition(position, offset);
    // convert the line/col position to a numeric offset
    const offsetOfPosition = this.getText()
      .split("\n")
      .reduce((acc, row, i) => {
        if (i === adjustedPosition.line) {
          return acc + adjustedPosition.character;
        } else if (i < adjustedPosition.line) return acc + row.length;
        else return acc;
      }, 0);
    let previousRange = [-Infinity, Infinity];
    YAML.visit(this.#root, {
      Node: (_, node) => {
        let n: YAML.Node | undefined = node;
        if (YAML.isAlias(node)) {
          n = node.resolve(this.#document);
        }
        if (n?.range) {
          const [start, end] = n.range;
          previousRange = n.range;
          const nodeContainsOffset = start <= offsetOfPosition &&
            end >= offsetOfPosition;
          const isSmallerThanPreviousRange = start >= previousRange[0] &&
            end <= previousRange[1];
          if (nodeContainsOffset && isSmallerThanPreviousRange) {
            found = node;
          }
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
    if (!YAML.isScalar(stringNode)) return null;
    if (typeof stringNode.value === "string") {
      const valueRange = this.#getRangeForNode(stringNode);

      if (!valueRange) return null;

      const valueLines = this.getText()
        .split("\n")
        .slice(valueRange.start.line, valueRange.end.line + 1);

      // get the match that contains the current position
      for (const currentLine of valueLines) {
        const matches = currentLine.match(REF_RE); // ['{color.blue._}', '{color.blue.dark}']
        for (const reference of matches ?? []) {
          const matchOffsetInLine = currentLine.indexOf(reference);
          if (
            adjusted.character >= matchOffsetInLine &&
            adjusted.character <= matchOffsetInLine + reference.length
          ) {
            const stringRange = this.#getRangeForNode(stringNode)!;
            const refUnpacked = reference.replace(REF_RE, "$1");
            const line = stringRange.start.line;
            const character = currentLine.indexOf(refUnpacked);
            const prefix = this.#context.workspaces.getPrefixForUri(this.uri);
            const path = [prefix, ...refUnpacked.split(".")].filter((x) => !!x);
            const name = `--${path.join("-")}`;
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

  public getColors(context: DTLSContext): LSP.ColorInformation[] {
    const colors: LSP.ColorInformation[] = [];
    YAML.visit(this.#root, {
      Scalar: (_key, node, path) => {
        const content = node.value;
        if (typeof content === "string") {
          const type = this.#getDTCGTypeForNode(node, path);
          if (type === "color" && node.range) {
            const range = this.#getRangeForNode(node);
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
      },
    });
    return colors;
  }

  public getDiagnostics(context: DTLSContext) {
    if (!context.tokens) {
      throw new Error("No tokens found in context");
    }
    const diagnostics: LSP.Diagnostic[] = [];
    YAML.visit(this.#root, {
      Scalar: (_, node) => {
        const range = this.#getRangeForNode(node);
        const content = node.value;
        if (!range || typeof content !== "string") return;
        const errors: string[] = [];
        if (usesReferences(content)) {
          const matches = content.match(REF_RE);
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
}
