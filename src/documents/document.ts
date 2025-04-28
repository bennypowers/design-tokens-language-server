import * as LSP from "vscode-languageserver-protocol";
import { FullTextDocument } from "./textDocument.ts";
import { Token } from "style-dictionary";
import { DTLSContext } from "#lsp/lsp.ts";

export abstract class DTLSTextDocument extends FullTextDocument {
  abstract language: "json" | "css";

  abstract getDiagnostics(context: DTLSContext): LSP.Diagnostic[];
  abstract getColors(context: DTLSContext): LSP.ColorInformation[];

  abstract getHoverTokenAtPosition(
    position: LSP.Position,
    offset?: Partial<LSP.Position>,
  ): {
    name: string;
    token: Token;
    range: LSP.Range;
  } | null;

  abstract definition(
    params: LSP.DefinitionParams,
    context: DTLSContext,
  ): LSP.Location[];

  get identifier(): LSP.VersionedTextDocumentIdentifier {
    return {
      uri: this.uri,
      version: this.version,
    };
  }

  #startOfSubstring(substring: string) {
    const text = this.getText();
    // get the position of the string in doc
    const rows = text.split("\n");
    const line = rows.findIndex((line) => line.includes(substring));
    const row = rows[line];
    if (row == null) {
      throw new Error(`Could not find string "${substring}" in document`);
    }
    const character = row.indexOf(substring);
    return { line, character };
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
    let { line, character } = this.#startOfSubstring(substring);
    if (position === "end") {
      character += substring.length;
    }
    return { line, character };
  }

  /**
   * Get the first range of the string in the document
   *
   * @param substring - The string to find in the document
   * @returns The range of the string in the document
   */
  getRangeForSubstring(substring: string): LSP.Range {
    const { line, character } = this.#startOfSubstring(substring);
    return {
      start: { line, character },
      end: { line, character: character + substring.length },
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
