import * as LSP from "vscode-languageserver-protocol";

import { adjustPosition, DTLSTextDocument, isPositionInRange } from "#document";

import { DTLSContext, DTLSErrorCodes } from "#lsp/lsp.ts";

import { Token } from "style-dictionary";

import * as YAML from "yaml";

import { cssColorToLspColor } from "#color";
import { usesReferences } from "style-dictionary/utils";
import { getLightDarkValues } from "#css";
import { Logger } from "#logger";

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
    const [startOffset, valueEndOffset, nodeEndOffset] = node.range;
    const startPos = this.#lineCounter.linePos(startOffset);
    const endPos = this.#lineCounter.linePos(valueEndOffset);
    return {
      start: { line: startPos.line, character: startPos.col },
      end: { line: endPos.line, character: endPos.col },
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
        typeNode = parent.get("$type");
        type = YAML.isScalar<string>(typeNode) ? typeNode.value : undefined;
      }
    }
    return type;
  }

  #getNodeAtPath(
    path: (string | number)[],
  ): YAML.Scalar | YAML.YAMLMap | YAML.YAMLSeq | null {
    const node = this.#document.getIn(path);
    if (YAML.isAlias(node)) {
      return node.resolve(this.#document) ?? null;
    } else if (YAML.isNode(node) && !YAML.isAlias(node)) {
      return node;
    } else {
      Logger.debug`Unknown ${node}`;
    }
    return null;
  }

  #getNodeAtPosition(
    position: LSP.Position,
    offset: Partial<LSP.Position> = {},
  ): YAML.Scalar | YAML.YAMLMap | YAML.YAMLSeq | null {
    let found: YAML.Node | null = null;
    const adjustedPosition = adjustPosition(position, offset);
    YAML.visit(this.#root, {
      Node: (_, node) => {
        let n: YAML.Node | undefined = node;
        if (YAML.isAlias(node)) {
          n = node.resolve(this.#document);
        }
        if (n) {
          const range = this.#getRangeForNode(n);
          if (range && isPositionInRange(adjustedPosition, range)) {
            found = node;
            return YAML.visit.BREAK;
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

  #getTokenForPath(path: (string | number)[]): Token | null {
    const node = this.#getNodeAtPath(path);
    const valueNode = this.#getNodeAtPath([...path, "$value"]);
    const descriptionNode = this.#getNodeAtPath([...path, "$description"]);
    if (!YAML.isScalar(descriptionNode)) {
      throw new Error("$description is not a string");
    }
    if (!node || !valueNode) {
      return null;
    }
    const $type = this.#getDTCGTypeForNode(valueNode);
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

    let $value = getValues(valueNode).join(""); // XXX: this may come back to bite me
    const $description = typeof descriptionNode?.value === "string"
      ? descriptionNode.value
      : undefined;
    if ($value) {
      if (usesReferences($value)) {
        const resolved = this.#context.tokens.resolveValue($value)?.toString();
        if (resolved) {
          $value = resolved;
        }
      }
      return { $value, $type, $description };
    }
    return null;
  }

  #getStringAtPosition(position: LSP.Position) {
    const node = this.#getNodeAtPosition(position);
    if (YAML.isScalar(node) && typeof node.value === "string") {
      return node.value;
    }
    return null;
  }

  getColors(context: DTLSContext): LSP.ColorInformation[] {
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

  getRangeForTokenName(tokenName: string, prefix?: string): LSP.Range | null {
    return this.#getRangeForNode(this.#getNodeForTokenName(tokenName, prefix));
  }

  getRangeForPath(path: (string | number)[]): LSP.Range | null {
    return this.#getRangeForNode(this.#getNodeAtPath(path));
  }

  getHoverTokenAtPosition(
    position: LSP.Position,
    offset: Partial<LSP.Position> = {},
  ) {
    const stringNode = this.#getNodeAtPosition(position, offset);
    if (!YAML.isScalar(stringNode)) return null;
    if (typeof stringNode.value === "string") {
      const valueRange = this.#getRangeForNode(stringNode);

      if (!valueRange) return null;
      const valueLines = this.getText().split("\n").slice(
        valueRange.start.line,
        valueRange.end.line,
      );

      const matches = `${stringNode.value}`.match(REF_RE); // ['{color.blue._}', '{color.blue.dark}']

      if (!matches) return null;

      // get the match that contains the current position
      for (const valueLine of valueLines) {
        for (const match of matches) {
          const matchOffsetInLine = valueLine.indexOf(match);
          if (
            position.character >= matchOffsetInLine &&
            position.character <= matchOffsetInLine + match.length
          ) {
            const name = match;
            const stringRange = this.#getRangeForNode(stringNode)!;
            const nameWithoutBraces = name.replace(REF_RE, "$1");
            const line = stringRange.start.line;
            const character = valueLine.indexOf(nameWithoutBraces);
            const path = nameWithoutBraces.split(".");
            const range = {
              start: { line, character },
              end: { line, character: character + name.length - 2 },
            };
            const tokenNode = this.#document.getIn(path);
            const token = this.#getTokenForPath(path);
            if (tokenNode && token && range) {
              return { name, range, token };
            } else {
              for (const referree of this.#context.documents.getAll("yaml")) {
                const token = referree.#getTokenForPath(path);
                if (token) {
                  return { name, range, token, path };
                }
              }
            }
          }
        }
      }
    }
    return null;
  }

  definition(params: LSP.DefinitionParams, context: DTLSContext) {
    const reference = this.#getStringAtPosition(params.position);
    const path = reference?.replace(/{|}/g, "").split(".");
    if (!path) return [];
    const node = this.#getNodeAtPath(path);
    if (node) {
      const range = this.#getRangeForNode(node);
      if (range) {
        return [{ uri: this.uri, range }];
      }
    } else {
      for (const referree of context.documents.getAll("yaml")) {
        const token = referree.#getTokenForPath(path);
        if (token) {
          const range = referree.#getRangeForNode(
            referree.#getNodeAtPath(path),
          );
          if (range) {
            return [{ uri: referree.uri, range }];
          }
        }
      }
    }
    return [];
  }

  getDiagnostics(context: DTLSContext) {
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
        diagnostics.push(...errors.map((name) => ({
          range,
          severity: LSP.DiagnosticSeverity.Error,
          message: `Token reference does not exist: ${name}`,
          code: DTLSErrorCodes.unknownReference,
        })));
      },
    });
    return diagnostics;
  }
}
