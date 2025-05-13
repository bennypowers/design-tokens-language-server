import * as LSP from "vscode-languageserver-protocol";
import { FullTextDocument } from "./textDocument.ts";
import { DTLSContext } from "#lsp/lsp.ts";
import { DTLSToken } from "#tokens";
import { Token } from "style-dictionary";

export const TOKEN_REFERENCE_REGEXP = /{(?<reference>[^}]+)}/gd;

/**
 * Represents an occurrence of a token name in a file
 */
export interface TokenReference {
  /**
   * The referenced token name.
   *
   * Not necessarily the same as the file substring,
   * e.g. `{color.red}` might reference a name `--token-color-red`
   */
  name: string;
  /**
   * The range in the document which refers to the token
   * Not necessarily the range for the referenced token name
   * e.g. the range for `{color.red}`, not `--token-color-red`
   */
  range: LSP.Range;
}

type Offset = Partial<LSP.Position>;

export abstract class DTLSTextDocument extends FullTextDocument {
  static isDTLSToken(token: Token): token is DTLSToken {
    return !!token?.$extensions?.designTokensLanguageServer;
  }

  abstract language: "json" | "css" | "yaml";

  /**
   * Get diagnostics for the document
   */
  abstract getDiagnostics(context: DTLSContext): LSP.Diagnostic[];

  /**
   * Get a list of ColorInformation objects representing the occurences of a
   * color token name in the document.
   */
  abstract getColors(context: DTLSContext): LSP.ColorInformation[];

  /**
   * Get a list of DocumentSymbol objects representing the occurences of tokens in the document.
   */
  abstract getDocumentSymbols(context: DTLSContext): LSP.DocumentSymbol[];

  /**
   * Computes token completions for a given position in a document
   */
  abstract getCompletions(
    context: DTLSContext,
    params: LSP.CompletionParams,
  ): LSP.CompletionList | null;

  /**
   * Get a range and the referenced token name for a position in a document
   * e.g. `{color.red}` => the range for `{color.red}` and the token name `--token-color-red`
   */
  abstract getTokenReferenceAtPosition(
    position: LSP.Position,
    offset?: Offset,
  ): TokenReference | null;

  /**
   * Gets the range of a data path in a JSON or YAML file
   */
  abstract getRangeForPath(path: string[]): LSP.Range | null;

  /**
   * Gets the token corresponding to a data path in a JSON or YAML file
   */
  abstract getTokenForPath(path: string[]): DTLSToken | null;

  /**
   * The identifier for this document
   */
  get identifier(): LSP.VersionedTextDocumentIdentifier {
    return {
      uri: this.uri,
      version: this.version,
    };
  }

  /**
   * Get the first range of the string in the document
   *
   * @param substring - The string to find in the document
   * @returns The range of the string in the document
   */
  public getRangeForSubstring(substring: string): LSP.Range {
    const [range] = this.getRangesForSubstring(substring);
    return range;
  }

  public getRangesForSubstring(substring: string): LSP.Range[] {
    const text = this.getText();
    const rows = text.split("\n");
    return rows
      .flatMap((row, line) => {
        const character = row.indexOf(substring);
        if (character < 0) return [];
        return [{
          start: { line, character },
          end: { line, character: character + substring.length },
        }];
      });
  }

  /**
   * Get definition for a token referenced in a document
   */
  public getDefinition(
    params: LSP.DefinitionParams,
    context: DTLSContext,
  ): LSP.Location | null {
    const ref = this.getTokenReferenceAtPosition(params.position);
    if (!ref) return null;
    const { name } = ref;
    const token = context.tokens.get(name);
    if (!token) return null;
    const ext = token.$extensions.designTokensLanguageServer;
    if (ext.definitionUri) {
      const doc = context.documents.get(ext.definitionUri);
      const uri = ext.definitionUri;
      const realPath = ext.groupMarker
        ? [...ext.path, ext.groupMarker]
        : ext.path;
      const range = doc.getRangeForPath(realPath);
      if (range) {
        return { uri, range };
      }
    }
    return null;
  }

  /** Get the range of the full document */
  public getFullRange() {
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
