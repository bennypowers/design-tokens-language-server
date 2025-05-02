import * as LSP from "vscode-languageserver-protocol";

import { DTLSContext, DTLSErrorCodes } from "#lsp/lsp.ts";

import { usesReferences } from "style-dictionary/utils";
import * as JSONC from "jsonc-parser";

import { cssColorToLspColor } from "#color";
import { getLightDarkValues } from "#css";
import { Logger } from "#logger";
import { DTLSTextDocument } from "#document";

import { DEFAULT_GROUP_MARKERS, DTLSToken } from "#tokens";

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

  private constructor(
    uri: string,
    version: number,
    text: string,
  ) {
    super(uri, "json", version, text);
    this.#root = this.#parse();
  }

  override update(
    changes: LSP.TextDocumentContentChangeEvent[],
    version: number,
  ) {
    super.update(changes, version);
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

  getTokenForPath(path: JSONC.Segment[]): DTLSToken | null {
    const node = this.#getNodeAtJSONPath(path);
    if (!node) {
      return null;
    }
    const valueNode = JSONC.findNodeAtLocation(node, ["$value"]);
    let startingNode = node;
    let typeNode = JSONC.findNodeAtLocation(node, ["$type"]);
    while (!typeNode && startingNode?.parent) {
      startingNode = startingNode.parent;
      typeNode = JSONC.findNodeAtLocation(startingNode, ["$type"]);
    }
    let $value = valueNode?.value;
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

  getRangeForPath(path: JSONC.Segment[]): LSP.Range | null {
    const node = this.#getNodeAtJSONPath(path);
    return this.#getRangeForNode(node);
  }

  getTokenReferenceAtPosition(
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
        const reference = matches?.find((match) => {
          const matchOffsetInLine = currentLine.indexOf(match);
          return (
            position.character >= matchOffsetInLine &&
            position.character <= matchOffsetInLine + match.length
          );
        });

        if (reference) {
          const stringRange = this.#getRangeForNode(stringNode)!;
          const refUnpacked = reference.replace(REF_RE, "$1");
          const line = stringRange.start.line;
          const character = currentLine.indexOf(refUnpacked);
          const spec = this.#context.workspaces.getSpecForUri(this.uri);
          const parts = refUnpacked.split(".");
          const pathIncludingMarkers = spec?.prefix
            ? [spec.prefix, ...parts]
            : parts;
          const groupMarkers = spec?.groupMarkers ?? DEFAULT_GROUP_MARKERS;
          const path = pathIncludingMarkers.filter((x) =>
            !groupMarkers.includes(x)
          );
          const name = `--${path.join("-")}`;
          Logger.debug`name:${name} ${this.#context.tokens.get(name)}`;
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
    return null;
  }

  getPathAtPosition(
    position: LSP.Position,
    offset: Partial<LSP.Position> = {},
  ): JSONC.Segment[] | null {
    const node = this.#getNodeAtPosition(position, offset);
    return node && JSONC.getNodePath(node);
  }

  public getColors(context: DTLSContext): LSP.ColorInformation[] {
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
        const resolved = context.tokens.resolveValue(content);
        const match = resolved?.toString() ?? content;
        const color = cssColorToLspColor(match);
        if (color) {
          colors.push({ range, color });
        }
      }
    }
    return colors;
  }

  public getDiagnostics(context: DTLSContext) {
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
          const resolved = context.tokens.resolveValue(name);
          if (!resolved) {
            errors.push(name);
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
