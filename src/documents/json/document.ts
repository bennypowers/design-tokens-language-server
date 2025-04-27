import * as LSP from "vscode-languageserver-protocol";

import { DTLSTextDocument } from "#document";

import { DTLSContext } from "#lsp";

import {
  findNodeAtLocation,
  type Node,
  parseTree,
  Segment,
} from "npm:jsonc-parser";

import { cssColorToLspColor } from "#color";
import { usesReferences } from "style-dictionary/utils";
import { getLightDarkValues } from "#css";

function offsetToPosition(content: string, offset: number): LSP.Position {
  const lines = content.split("\n");
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

export class JsonDocument extends DTLSTextDocument {
  language = "json" as const;

  #root: Node;
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
    const getTypeColorValues = (node: Node) => {
      const valueNode = findNodeAtLocation(node, ["$value"]);
      const content = valueNode?.value;
      if (valueNode && typeof content === "string") {
        const range = this.#getRangeForNode(valueNode)!;
        if (usesReferences(content)) {
          const references = content.match(/{[^}]*}/g);
          for (const reference of references ?? []) {
            const resolved = context.tokens.resolve(reference);
            if (resolved) {
              const line = range.start.line;
              const character = range.start.character +
                content.indexOf(reference) + 2;
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
    const getColors = (node: Node) => {
      if (findNodeAtLocation(node, ["$type"])?.value === "color") {
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

  #parse(): Node {
    const root = parseTree(this.getText());
    if (!root) {
      throw new Error("Failed to parse JSON");
    }
    return root;
  }

  #getNodeAtJSONPath(path: Segment[]): Node | null {
    return findNodeAtLocation(this.#root, path) ?? null;
  }

  #getNodeForTokenName(tokenName: string, prefix?: string): Node | null {
    const tokenPath = tokenName.replace(/^--/, "")
      .split("-")
      .filter((x) => !!x)
      .filter((x) => prefix ? x !== prefix : true);
    const node = this.#getNodeAtJSONPath(tokenPath);
    return node ?? null;
  }

  #getRangeForNode(node: Node | null): LSP.Range | null {
    if (node) {
      const start = node.offset;
      const end = start + node.length;
      const content = this.getText();

      return {
        start: offsetToPosition(content, start),
        end: offsetToPosition(content, end),
      };
    }
    return null;
  }

  getRangeForTokenName(tokenName: string, prefix?: string): LSP.Range | null {
    return this.#getRangeForNode(this.#getNodeForTokenName(tokenName, prefix));
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
