import * as LSP from "vscode-languageserver-protocol";
import type { DTLSContext } from "#lsp";
import { FullTextDocument } from "./textDocument.ts";

export abstract class DTLSTextDocument extends FullTextDocument {
  diagnostics: LSP.Diagnostic[] = [];
  abstract language: "json" | "css";
  abstract computeDiagnostics(_: DTLSContext): LSP.Diagnostic[];

  get identifier(): LSP.VersionedTextDocumentIdentifier {
    return {
      uri: this.uri,
      version: this.version,
    };
  }

  /**
   * Get the first position of the string in the document
   *
   * @param substring - The string to find in the document
   * @param position - The position in the substring to return (start or end)
   * @returns The position of the start or end of the string in the document
   */
  positionForSubstring(
    substring: string,
    position: "start" | "end" = "start",
  ): LSP.Position {
    const text = this.getText();
    // get the position of the string in doc
    const rows = text.split("\n");
    const line = rows.findIndex((line) => line.includes(substring));
    let character = rows[line].indexOf(substring);
    if (position === "end") {
      character += substring.length;
    }
    return { line, character };
  }

  /**
   * Get the first range of the string in the document
   *
   * @param string - The string to find in the document
   * @returns The range of the string in the document
   */
  rangeForSubstring(string: string): LSP.Range {
    const text = this.getText();
    // get the range of the string in doc
    const rows = text.split("\n");
    const line = rows.findIndex((line) => line.includes(string));
    const character = rows[line].indexOf(string);
    return {
      start: { line, character },
      end: { line, character: character + string.length },
    };
  }

  /** Get the range of the full document */
  fullRange() {
    const text = this.getText();
    // get the range of the string in doc
    const rows = text.split("\n");
    const line = rows.length - 1;
    const character = rows[line].length;
    return {
      start: { line: 0, character: 0 },
      end: { line, character },
    };
  }
}
