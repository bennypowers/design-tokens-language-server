import * as LSP from "vscode-languageserver-protocol";

import { DTLSTextDocument } from "#document";

import { DTLSContext } from "#lsp";

import {
  findNodeAtLocation,
  type Node,
  parseTree,
  Segment,
} from "npm:jsonc-parser";

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
  #diagnostics: LSP.Diagnostic[] = [];

  get diagnostics() {
    return this.#diagnostics;
  }

  static create(
    _context: DTLSContext,
    uri: string,
    text: string,
    version = 0,
  ) {
    const doc = new JsonDocument(uri, version, text);
    doc.#diagnostics = doc.#computeDiagnostics();
    return doc;
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

  getRangeForTokenName(tokenName: string, prefix?: string): LSP.Range | null {
    const node = this.#getNodeForTokenName(tokenName, prefix);
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
