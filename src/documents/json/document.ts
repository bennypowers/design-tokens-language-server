import * as LSP from "vscode-languageserver-protocol";

import { DTLSTextDocument } from "#document";

import { DTLSContext } from "#lsp/lsp.ts";

import * as JSONC from "npm:jsonc-parser";

import { cssColorToLspColor } from "#color";
import { usesReferences } from "style-dictionary/utils";
import { getLightDarkValues } from "#css";
import { Token } from "style-dictionary";

export class JsonDocument extends DTLSTextDocument {
  language = "json" as const;

  #root: JSONC.Node;
  #context!: DTLSContext;
  #diagnostics: LSP.Diagnostic[] = [];

  static create(
    context: DTLSContext,
    uri: string,
    text: string,
    version = 0,
  ) {
    const doc = new JsonDocument(uri, version, text);
    doc.#diagnostics = doc.#computeDiagnostics();
    doc.#context = context;
    return doc;
  }

  get diagnostics() {
    return this.#diagnostics;
  }

  get colors(): LSP.ColorInformation[] {
    const colors: LSP.ColorInformation[] = [];
    const context = this.#context;
    const getTypeColorValues = (node: JSONC.Node) => {
      const valueNode = JSONC.findNodeAtLocation(node, ["$value"]);
      const content = valueNode?.value;
      if (valueNode && typeof content === "string") {
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
            const resolved = context.tokens.resolve(reference);
            if (resolved) {
              const line = range.start.line;
              const character = range.start.character +
                content.indexOf(reference) + 1;
              colors.push({
                color: cssColorToLspColor(resolved.toString()),
                range: {
                  start: { line, character },
                  end: { line, character: character + reference.length - 2 },
                },
              });
            }
          }
        } else if (content.startsWith("light-dark(")) {
          const [light, dark] = getLightDarkValues(content);
          colors.push({
            range,
            color: cssColorToLspColor(light),
          });
          colors.push({
            range,
            color: cssColorToLspColor(dark),
          });
        } else {
          colors.push({
            range,
            color: cssColorToLspColor(
              context.tokens.resolve(content)?.toString() ?? content,
            ),
          });
        }
      }
      node.children?.forEach(getTypeColorValues);
    };
    const getColors = (node: JSONC.Node) => {
      if (JSONC.findNodeAtLocation(node, ["$type"])?.value === "color") {
        getTypeColorValues(node);
      }
      node.children?.forEach(getColors);
    };
    this.#root?.children?.forEach(getColors);
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
    const $value = valueNode?.value;
    const $type = typeNode?.value;
    const $description = descriptionNode?.value;
    if ($value) {
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

  getTokenAtPosition(
    position: LSP.Position,
    offset: Partial<LSP.Position> = {},
  ) {
    const REF_RE = /{([^}]+)}/g;
    const node = this.#getNodeAtPosition(position, offset);
    switch (node?.type) {
      case "string": {
        const matches = `${node.value}`.match(REF_RE); // ['{color.blue._}', '{color.blue.dark}']
        // because it's json, we only need to check the current line
        const line = this.getText().split("\n")[position.line]; // "$value": "light-dark({color.blue._}, {color.blue.dark})"

        // get the match that contains the current position
        const match = matches?.find((match) => {
          const matchOffsetInLine = line.indexOf(match);
          return (
            position.character >= matchOffsetInLine &&
            position.character <= matchOffsetInLine + match.length
          );
        });

        if (match) {
          const path = match.replace(REF_RE, "$1").split(".");
          const node = JSONC.findNodeAtLocation(this.#root, path) ?? null;
          const token = this.#getTokenForPath(path);
          if (node && token) {
            return {
              name: match,
              range: this.#getRangeForNode(node)!,
              token,
            };
          }
        }
        return null;
      }
      default:
        return null;
    }
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
    this.#diagnostics = this.#computeDiagnostics();
  }

  #computeDiagnostics(): LSP.Diagnostic[] {
    return [];
  }
}
