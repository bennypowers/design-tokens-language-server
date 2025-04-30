import * as LSP from "vscode-languageserver-protocol";
import { FullTextDocument } from "./textDocument.ts";
import { Token } from "style-dictionary";
import { DTLSContext } from "#lsp/lsp.ts";

export abstract class DTLSTextDocument extends FullTextDocument {
  abstract language: "json" | "css" | "yaml";

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

  #startOfSubstrings(substring: string) {
    const text = this.getText();
    const rows = text.split("\n");
    return rows
      .filter((line) => line.includes(substring))
      .map((row) => {
        const line = rows.indexOf(row);
        const character = row.indexOf(substring);
        return { line, character };
      });
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
    let [{ line, character }] = this.#startOfSubstrings(substring);
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
    const [range] = this.getRangesForSubstring(substring);
    return range;
  }

  getRangesForSubstring(substring: string): LSP.Range[] {
    return this.#startOfSubstrings(substring).map(({ line, character }) => {
      return {
        start: { line, character },
        end: { line, character: character + substring.length },
      };
    });
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

export function isPositionInRange(position: LSP.Position, range: LSP.Range) {
  const { start, end } = range;
  return (
    position.line >= start.line &&
    position.line <= end.line &&
    (position.line === start.line
      ? position.character >= start.character
      : true) &&
    (position.line === end.line ? position.character <= end.character : true)
  );
}

export function adjustPosition(
  position: LSP.Position,
  offset?: Partial<LSP.Position>,
) {
  if (!offset) return position;
  return {
    line: position.line + (offset.line ?? 0),
    character: position.character + (offset.character ?? 0),
  };
}
