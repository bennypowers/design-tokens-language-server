import * as LSP from "vscode-languageserver-protocol";

import { DTLSTextDocument } from "#document";

import { DTLSContext, DTLSErrorCodes } from "#lsp/lsp.ts";

import * as JSONC from "npm:jsonc-parser";

import { cssColorToLspColor } from "#color";
import { usesReferences } from "style-dictionary/utils";
import { getLightDarkValues } from "#css";
import { Token } from "style-dictionary";

const REF_RE = /{([^}]+)}/g;

export class JsonDocument extends DTLSTextDocument {
  language = "json" as const;

  #root: JSONC.Node;
  #context!: DTLSContext;

  static create(
    context: DTLSContext,
    uri: string,
    text: string,
    version = 0,
  ) {
    const doc = new JsonDocument(uri, version, text);
    doc.#context = context;
    return doc;
  }

  get #allStringValueNodes() {
    const nodes: JSONC.Node[] = [];
    const getStringValueNodesInNode = (node: JSONC.Node) => {
      const valueNode = JSONC.findNodeAtLocation(node, ["$value"]);
      const content = valueNode?.value;
      if (valueNode && typeof content === "string") {
        nodes.push(valueNode);
      }
      node.children?.forEach(getStringValueNodesInNode);
    };
    this.#root?.children?.forEach(getStringValueNodesInNode);
    return nodes;
  }

  getColors(context: DTLSContext): LSP.ColorInformation[] {
    const colors: LSP.ColorInformation[] = [];
    for (const valueNode of this.#allStringValueNodes) {
      // cheap hack to avoid marking up non-color values
      // a more comprehensive solution would be to associate each json document
      // with a token spec, and get the token object *with prefix* from the context
      // based on the *non-prefixed* json path, then check the type from the token object
      let parent = valueNode.parent;
      let type = JSONC.findNodeAtLocation(parent!, ["$type"])?.value;
      while (!type && parent) {
        parent = parent.parent;
        if (parent) {
          type = JSONC.findNodeAtLocation(parent, ["$type"])?.value;
        }
      }
      if (type && type !== "color") {
        continue;
      }
      const content = valueNode.value;
      const _range = this.#getRangeForNode(valueNode)!;
      const range = {
        start: {
          line: _range.start.line,
          character: _range.start.character + 1,
        },
        end: {
          line: _range.end.line,
          character: _range.end.character - 1,
        },
      };
      if (usesReferences(content)) {
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
                  end: { line, character: character + reference.length - 2 },
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
        const match = context.tokens.resolveValue(content)?.toString() ??
          content;
        const color = cssColorToLspColor(match);
        if (color) {
          colors.push({ range, color });
        }
      }
    }
    return colors;
  }

  private constructor(
    uri: string,
    version: number,
    text: string,
  ) {
    super(uri, "json", version, text);
    this.#root = this.#parse();
  }

  #parse(): JSONC.Node {
    const root = JSONC.parseTree(this.getText());
    if (!root) {
      throw new Error("Failed to parse JSON");
    }
    return root;
  }

  #positionToOffset(position: LSP.Position): number {
    const lines = this.getText().split("\n");
    let offset = 0;

    for (let i = 0; i < position.line; i++) {
      offset += lines[i].length + 1; // +1 for the newline character
    }

    offset += position.character;
    return offset;
  }

  #offsetToPosition(offset: number): LSP.Position {
    const lines = this.getText().split("\n");
    let line = 0;
    let column = offset;

    for (let i = 0; i < lines.length; i++) {
      if (column <= lines[i].length) {
        line = i;
        break;
      }
      column -= lines[i].length + 1; // +1 for the newline character
    }

    return { line, character: column };
  }

  #getNodeAtJSONPath(path: JSONC.Segment[]): JSONC.Node | null {
    return JSONC.findNodeAtLocation(this.#root, path) ?? null;
  }

  #getNodeAtPosition(
    position: LSP.Position,
    offset: Partial<LSP.Position> = {},
  ): JSONC.Node | null {
    return JSONC.findNodeAtOffset(
      this.#root,
      this.#positionToOffset({
        line: position.line + (offset.line ?? 0),
        character: position.character + (offset.character ?? 0),
      }),
    ) ?? null;
  }

  #getNodeForTokenName(tokenName: string, prefix?: string): JSONC.Node | null {
    const tokenPath = tokenName.replace(/^--/, "")
      .split("-")
      .filter((x) => !!x)
      .filter((x) => prefix ? x !== prefix : true);
    const node = this.#getNodeAtJSONPath(tokenPath);
    return node ?? null;
  }

  #getRangeForNode(node: JSONC.Node | null): LSP.Range | null {
    if (node) {
      const start = node.offset;
      const end = start + node.length;

      return {
        start: this.#offsetToPosition(start),
        end: this.#offsetToPosition(end),
      };
    }
    return null;
  }

  #getTokenForPath(path: JSONC.Segment[]): Token | null {
    const node = this.#getNodeAtJSONPath(path);
    if (!node) {
      return null;
    }
    const valueNode = JSONC.findNodeAtLocation(node, ["$value"]);
    const descriptionNode = JSONC.findNodeAtLocation(node, ["$description"]);
    let startingNode = node;
    let typeNode = JSONC.findNodeAtLocation(node, ["$type"]);
    while (!typeNode && startingNode?.parent) {
      startingNode = startingNode.parent;
      typeNode = JSONC.findNodeAtLocation(startingNode, ["$type"]);
    }
    let $value = valueNode?.value;
    const $type = typeNode?.value;
    const $description = descriptionNode?.value;
    if ($value) {
      if (usesReferences($value)) {
        $value = this.#context.tokens.resolveValue($value)?.toString();
      }
      return { $value, $type, $description };
    }
    return null;
  }

  getRangeForTokenName(tokenName: string, prefix?: string): LSP.Range | null {
    return this.#getRangeForNode(this.#getNodeForTokenName(tokenName, prefix));
  }

  getRangeForPath(path: JSONC.Segment[]): LSP.Range | null {
    return this.#getRangeForNode(this.#getNodeAtJSONPath(path));
  }

  getHoverTokenAtPosition(
    position: LSP.Position,
    offset: Partial<LSP.Position> = {},
  ) {
    const stringNode = this.#getNodeAtPosition(position, offset);
    switch (stringNode?.type) {
      case "string": {
        const matches = `${stringNode.value}`.match(REF_RE); // ['{color.blue._}', '{color.blue.dark}']
        // because it's json, we only need to check the current line
        const currentLine = this.getText().split("\n")[position.line]; // "$value": "light-dark({color.blue._}, {color.blue.dark})"

        // get the match that contains the current position
        const name = matches?.find((match) => {
          const matchOffsetInLine = currentLine.indexOf(match);
          return (
            position.character >= matchOffsetInLine &&
            position.character <= matchOffsetInLine + match.length
          );
        });

        if (name) {
          const stringRange = this.#getRangeForNode(stringNode)!;
          const nameWithoutBraces = name.replace(REF_RE, "$1");
          const line = stringRange.start.line;
          const character = currentLine.indexOf(nameWithoutBraces);
          const path = nameWithoutBraces.split(".");
          const range = {
            start: { line, character },
            end: { line, character: character + name.length - 2 },
          };
          const tokenNode = JSONC.findNodeAtLocation(this.#root, path) ?? null;
          const token = this.#getTokenForPath(path);
          if (tokenNode && token && range) {
            return { name, range, token };
          } else {
            for (const referree of this.#context.documents.getAll("json")) {
              const token = referree.#getTokenForPath(path);
              if (token) {
                return { name, range, token, path };
              }
            }
          }
        }
        return null;
      }
      default:
        return null;
    }
  }

  getPathAtPosition(
    position: LSP.Position,
    offset: Partial<LSP.Position> = {},
  ): JSONC.Segment[] | null {
    const node = this.#getNodeAtPosition(position, offset);
    return node && JSONC.getNodePath(node);
  }

  #getStringAtPosition(position: LSP.Position) {
    const node = this.#getNodeAtPosition(position);
    if (node?.type === "string") {
      return node.value;
    }
    return null;
  }

  definition(
    params: LSP.DefinitionParams,
    context: DTLSContext,
  ) {
    const reference = this.#getStringAtPosition(params.position);
    const path = reference?.replace(/{|}/g, "").split(".");
    const node = this.#getNodeAtJSONPath(path);

    if (node) {
      const range = this.#getRangeForNode(node);
      if (range) {
        return [{ uri: this.uri, range }];
      }
    } else {for (const referree of context.documents.getAll("json")) {
        const token = referree.#getTokenForPath(path);
        if (token) {
          const range = referree.#getRangeForNode(
            referree.#getNodeAtJSONPath(path),
          );
          if (range) {
            return [{ uri: referree.uri, range }];
          }
        }
      }}
    return [];
  }

  override update(
    changes: LSP.TextDocumentContentChangeEvent[],
    version: number,
  ) {
    super.update(changes, version);
    this.#root = this.#parse();
  }

  getDiagnostics(context: DTLSContext) {
    if (!context.tokens) {
      throw new Error("No tokens found in context");
    }
    // all nodes which are string values of $value properties in the #root
    return this.#allStringValueNodes.flatMap((valueNode) => {
      const content = valueNode.value;
      const errors: string[] = [];
      if (usesReferences(content)) {
        const matches = content.match(REF_RE);
        for (const name of matches ?? []) {
          try {
            context.tokens.resolveValue(name);
          } catch (error) {
            if (error instanceof Error) {
              errors.push(name);
            } else throw error;
          }
        }
      }
      return errors.map((name) => ({
        range: this.#getRangeForNode(valueNode)!,
        severity: LSP.DiagnosticSeverity.Error,
        message: `Token reference does not exist: ${name}`,
        code: DTLSErrorCodes.unknownReference,
      }));
    });
  }
}
