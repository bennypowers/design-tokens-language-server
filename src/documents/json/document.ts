import * as LSP from "vscode-languageserver-protocol";

import { DTLSContext, DTLSErrorCodes } from "#lsp/lsp.ts";

import { usesReferences } from "style-dictionary/utils";
import * as JSONC from "jsonc-parser";

import { cssColorToLspColor } from "#color";
import { getLightDarkValues } from "#css";
import { DTLSTextDocument, TOKEN_REFERENCE_REGEXP } from "#document";

import { DEFAULT_GROUP_MARKERS, DTLSToken } from "#tokens";
import {
  DTLSSemanticTokenIntermediate,
  DTLSTokenTypes,
} from "#lsp/methods/textDocument/semanticTokens.ts";

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

  #localReferenceToTokenName(reference: string | (JSONC.Segment[])) {
    const prefix = this.#context.workspaces.getPrefixForUri(this.uri);
    const path = Array.isArray(reference) ? reference : reference.split(".");
    return [prefix, ...path].filter((x) => !!x).join("-");
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
      return this.#context.tokens.get(this.#localReferenceToTokenName(path)) ??
        null;
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
        const matches = `${stringNode.value}`.match(TOKEN_REFERENCE_REGEXP); // ['{color.blue._}', '{color.blue.dark}']
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
          const refUnpacked = reference.replace(TOKEN_REFERENCE_REGEXP, "$1");
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

  public getSemanticTokensFull(): DTLSSemanticTokenIntermediate[] {
    return this.#allStringValueNodes.flatMap((node) => {
      if (typeof node.value !== "string" || !usesReferences(node.value)) {
        return [];
      }
      const valueNodeRange = this.#getRangeForNode(node);
      if (!valueNodeRange) return [];
      const matches = node.value.matchAll(TOKEN_REFERENCE_REGEXP);
      if (!matches) return [];
      const line = valueNodeRange.start.line;
      return matches.flatMap((match) => {
        const { reference } = match.groups!;
        const name = this.#localReferenceToTokenName(reference);
        if (!this.#context.tokens.has(name)) return [];
        const [start] = match.indices!.groups!.reference;
        let lastStartChar = 1 + valueNodeRange.start.character + start;
        return reference.split(".").map((token, j) => {
          const startChar = lastStartChar;
          const { length } = token;
          const tokenModifiers = 0;
          const tokenType = DTLSTokenTypes[j] ?? DTLSTokenTypes[1];
          lastStartChar += length + 1;
          return {
            token,
            line,
            startChar,
            length,
            tokenType,
            tokenModifiers,
          };
        });
      }).toArray();
    });
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

  public getCompletions(
    context: DTLSContext,
    params: LSP.CompletionParams,
  ): LSP.CompletionList | null {
    const node = this.#getNodeAtPosition(params.position);
    if (node?.type === "string") {
      const prefix = node.value.replace(/}$/, "");
      const range = this.#getRangeForNode(node);
      if (range) {
        const items = context.tokens
          .values()
          .flatMap((token) => {
            const ext = token.$extensions.designTokensLanguageServer;
            if (!ext.reference.startsWith(prefix)) return [];
            const label = `"${ext.reference}"`;
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
    // all nodes which are string values of $value properties in the #root
    this.#allStringValueNodes.forEach((valueNode) => {
      const content: string = valueNode.value;
      if (usesReferences(content)) {
        const matches = content.match(TOKEN_REFERENCE_REGEXP);
        for (const match of matches ?? []) {
          const resolved = context.tokens.resolveValue(match);
          if (!resolved) {
            diagnostics.push({
              range: this.#getRangeForNode(valueNode)!,
              severity: LSP.DiagnosticSeverity.Error,
              message: `Token reference does not exist: ${match}`,
              code: DTLSErrorCodes.unknownReference,
            });
          } else {
            const refUnpacked = match.replace(TOKEN_REFERENCE_REGEXP, "$1");
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
            const token = context.tokens.get(name);
            if (token?.$deprecated) {
              let message = `${name} is deprecated`;
              if (typeof token.$deprecated === "string") {
                message += `: ${token.$deprecated}`;
              }
              diagnostics.push({
                range: this.#getRangeForNode(valueNode)!,
                severity: LSP.DiagnosticSeverity.Information,
                tags: [LSP.DiagnosticTag.Deprecated],
                message,
                data: {
                  tokenName: token.$extensions.designTokensLanguageServer.name,
                },
              });
            }
          }
        }
      }
    });
    return diagnostics;
  }
}
