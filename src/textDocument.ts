// deno-coverage-ignore-file
// See upstream for coverage

/* --------------------------------------------------------------------------------------------
 * Copyright (c) Microsoft Corporation. All rights reserved.
 * Licensed under the MIT License. See License.txt in the project root for license information.
 * ------------------------------------------------------------------------------------------ */

import * as LSP from "vscode-languageserver-protocol";

interface IncrementalEvent {
  range: LSP.Range;
  rangeLength?: number;
  text: string;
}

/**
 * A simple text document. Not to be implemented. The document keeps the content
 * as string.
 */
export interface TextDocument {
  /**
   * The associated URI for this document. Most documents have the __file__-scheme, indicating that they
   * represent files on disk. However, some documents may have other schemes indicating that they are not
   * available on disk.
   *
   * @readonly
   */
  readonly uri: LSP.DocumentUri;

  /**
   * The identifier of the language associated with this document.
   *
   * @readonly
   */
  readonly languageId: string;

  /**
   * The version number of this document (it will increase after each
   * change, including undo/redo).
   *
   * @readonly
   */
  readonly version: number;

  /**
	 * Get the text of this document. A substring can be retrieved by
	 * providing a range.
	 *
	 * @param range (optional) An range within the document to return.
	 * If no range is passed, the full content is returned.
	 * Invalid range positions are adjusted as described in {@link Position.line}
	 * and {@link Position.character}.
	 * If the start range position is greater than the end range position,
	 * then the effect of getText is as if the two positions were swapped.

	 * @return The text of this document or a substring of the text if a
	 *         range is provided.
	 */
  getText(range?: LSP.Range): string;

  /**
   * Converts a zero-based offset to a position.
   *
   * @param offset A zero-based offset.
   * @return A valid {@link Position position}.
   * @example The text document "ab\ncd" produces:
   * * position { line: 0, character: 0 } for `offset` 0.
   * * position { line: 0, character: 1 } for `offset` 1.
   * * position { line: 0, character: 2 } for `offset` 2.
   * * position { line: 1, character: 0 } for `offset` 3.
   * * position { line: 1, character: 1 } for `offset` 4.
   */
  positionAt(offset: number): LSP.Position;

  /**
   * Converts the position to a zero-based offset.
   * Invalid positions are adjusted as described in {@link Position.line}
   * and {@link Position.character}.
   *
   * @param position A position.
   * @return A valid zero-based offset.
   */
  offsetAt(position: LSP.Position): number;

  /**
   * The number of lines in this document.
   *
   * @readonly
   */
  readonly lineCount: number;
}
const enum CharCode {
  /**
   * The `\n` character.
   */
  LineFeed = 10,
  /**
   * The `\r` character.
   */
  CarriageReturn = 13,
}

function mergeSort<T>(data: T[], compare: (a: T, b: T) => number): T[] {
  if (data.length <= 1) {
    // sorted
    return data;
  }
  const p = (data.length / 2) | 0;
  const left = data.slice(0, p);
  const right = data.slice(p);

  mergeSort(left, compare);
  mergeSort(right, compare);

  let leftIdx = 0;
  let rightIdx = 0;
  let i = 0;
  while (leftIdx < left.length && rightIdx < right.length) {
    const ret = compare(left[leftIdx], right[rightIdx]);
    if (ret <= 0) {
      // smaller_equal -> take left to preserve order
      data[i++] = left[leftIdx++];
    } else {
      // greater -> take right
      data[i++] = right[rightIdx++];
    }
  }
  while (leftIdx < left.length) {
    data[i++] = left[leftIdx++];
  }
  while (rightIdx < right.length) {
    data[i++] = right[rightIdx++];
  }
  return data;
}

function computeLineOffsets(
  text: string,
  isAtLineStart: boolean,
  textOffset = 0,
): number[] {
  const result: number[] = isAtLineStart ? [textOffset] : [];
  for (let i = 0; i < text.length; i++) {
    const ch = text.charCodeAt(i);
    if (isEOL(ch)) {
      if (
        ch === CharCode.CarriageReturn && i + 1 < text.length &&
        text.charCodeAt(i + 1) === CharCode.LineFeed
      ) {
        i++;
      }
      result.push(textOffset + i + 1);
    }
  }
  return result;
}

function isEOL(char: number) {
  return char === CharCode.CarriageReturn || char === CharCode.LineFeed;
}

function getWellformedRange(range: LSP.Range): LSP.Range {
  const start = range.start;
  const end = range.end;
  if (
    start.line > end.line ||
    (start.line === end.line && start.character > end.character)
  ) {
    return { start: end, end: start };
  }
  return range;
}

function getWellformedEdit(textEdit: LSP.TextEdit): LSP.TextEdit {
  const range = getWellformedRange(textEdit.range);
  if (range !== textEdit.range) {
    return { newText: textEdit.newText, range };
  }
  return textEdit;
}

export class FullTextDocument implements TextDocument {
  /**
   * Updates a TextDocument by modifying its content.
   *
   * @param document the document to update. Only documents created by TextDocument.create are valid inputs.
   * @param changes the changes to apply to the document.
   * @param version the changes version for the document.
   * @returns The updated TextDocument. Note: That's the same document instance passed in as first parameter.
   */
  static update(
    document: TextDocument,
    changes: LSP.TextDocumentContentChangeEvent[],
    version: number,
  ): TextDocument {
    if (document instanceof FullTextDocument) {
      document.update(changes, version);
      return document;
    } else {
      throw new Error(
        "TextDocument.update: document must be created by TextDocument.create",
      );
    }
  }

  static applyEdits(document: TextDocument, edits: LSP.TextEdit[]): string {
    const text = document.getText();
    const sortedEdits = mergeSort(edits.map(getWellformedEdit), (a, b) => {
      const diff = a.range.start.line - b.range.start.line;
      if (diff === 0) {
        return a.range.start.character - b.range.start.character;
      }
      return diff;
    });
    let lastModifiedOffset = 0;
    const spans = [];
    for (const e of sortedEdits) {
      const startOffset = document.offsetAt(e.range.start);
      if (startOffset < lastModifiedOffset) {
        throw new Error("Overlapping edit");
      } else if (startOffset > lastModifiedOffset) {
        spans.push(text.substring(lastModifiedOffset, startOffset));
      }
      if (e.newText.length) {
        spans.push(e.newText);
      }
      lastModifiedOffset = document.offsetAt(e.range.end);
    }
    spans.push(text.substr(lastModifiedOffset));
    return spans.join("");
  }

  private _uri: LSP.DocumentUri;
  private _languageId: string;
  private _version: number;
  private _content: string;
  private _lineOffsets: number[] | undefined;

  public constructor(
    uri: LSP.DocumentUri,
    languageId: string,
    version: number,
    content: string,
  ) {
    this._uri = uri;
    this._languageId = languageId;
    this._version = version;
    this._content = content;
    this._lineOffsets = undefined;
  }

  public get uri(): string {
    return this._uri;
  }

  public get languageId(): string {
    return this._languageId;
  }

  public get version(): number {
    return this._version;
  }

  public getText(range?: LSP.Range): string {
    if (range) {
      const start = this.offsetAt(range.start);
      const end = this.offsetAt(range.end);
      return this._content.substring(start, end);
    }
    return this._content;
  }

  public update(
    changes: LSP.TextDocumentContentChangeEvent[],
    version: number,
  ): void {
    for (const change of changes) {
      if (FullTextDocument.isIncremental(change)) {
        // makes sure start is before end
        const range = getWellformedRange(change.range);

        // update content
        const startOffset = this.offsetAt(range.start);
        const endOffset = this.offsetAt(range.end);
        this._content = this._content.substring(0, startOffset) + change.text +
          this._content.substring(endOffset, this._content.length);

        // update the offsets
        const startLine = Math.max(range.start.line, 0);
        const endLine = Math.max(range.end.line, 0);
        let lineOffsets = this._lineOffsets!;
        const addedLineOffsets = computeLineOffsets(
          change.text,
          false,
          startOffset,
        );
        if (endLine - startLine === addedLineOffsets.length) {
          for (let i = 0, len = addedLineOffsets.length; i < len; i++) {
            lineOffsets[i + startLine + 1] = addedLineOffsets[i];
          }
        } else {
          if (addedLineOffsets.length < 10000) {
            lineOffsets.splice(
              startLine + 1,
              endLine - startLine,
              ...addedLineOffsets,
            );
          } else { // avoid too many arguments for splice
            this._lineOffsets = lineOffsets = lineOffsets.slice(
              0,
              startLine + 1,
            ).concat(addedLineOffsets, lineOffsets.slice(endLine + 1));
          }
        }
        const diff = change.text.length - (endOffset - startOffset);
        if (diff !== 0) {
          for (
            let i = startLine + 1 + addedLineOffsets.length,
              len = lineOffsets.length;
            i < len;
            i++
          ) {
            lineOffsets[i] = lineOffsets[i] + diff;
          }
        }
      } else if (FullTextDocument.isFull(change)) {
        this._content = change.text;
        this._lineOffsets = undefined;
      } else {
        throw new Error("Unknown change event received");
      }
    }
    this._version = version;
  }

  private getLineOffsets(): number[] {
    if (this._lineOffsets === undefined) {
      this._lineOffsets = computeLineOffsets(this._content, true);
    }
    return this._lineOffsets;
  }

  public positionAt(offset: number): LSP.Position {
    offset = Math.max(Math.min(offset, this._content.length), 0);

    const lineOffsets = this.getLineOffsets();
    let low = 0, high = lineOffsets.length;
    if (high === 0) {
      return { line: 0, character: offset };
    }
    while (low < high) {
      const mid = Math.floor((low + high) / 2);
      if (lineOffsets[mid] > offset) {
        high = mid;
      } else {
        low = mid + 1;
      }
    }
    // low is the least x for which the line offset is larger than the current offset
    // or array.length if no line offset is larger than the current offset
    const line = low - 1;

    offset = this.ensureBeforeEOL(offset, lineOffsets[line]);
    return { line, character: offset - lineOffsets[line] };
  }

  public offsetAt(position: LSP.Position) {
    const lineOffsets = this.getLineOffsets();
    if (position.line >= lineOffsets.length) {
      return this._content.length;
    } else if (position.line < 0) {
      return 0;
    }
    const lineOffset = lineOffsets[position.line];
    if (position.character <= 0) {
      return lineOffset;
    }

    const nextLineOffset = (position.line + 1 < lineOffsets.length)
      ? lineOffsets[position.line + 1]
      : this._content.length;
    const offset = Math.min(lineOffset + position.character, nextLineOffset);
    return this.ensureBeforeEOL(offset, lineOffset);
  }

  private ensureBeforeEOL(offset: number, lineOffset: number): number {
    while (offset > lineOffset && isEOL(this._content.charCodeAt(offset - 1))) {
      offset--;
    }
    return offset;
  }

  public get lineCount() {
    return this.getLineOffsets().length;
  }

  private static isIncremental(
    event: LSP.TextDocumentContentChangeEvent,
  ): event is { range: LSP.Range; rangeLength?: number; text: string } {
    const candidate: IncrementalEvent = event as IncrementalEvent;
    return candidate !== undefined && candidate !== null &&
      typeof candidate.text === "string" && candidate.range !== undefined &&
      (candidate.rangeLength === undefined ||
        typeof candidate.rangeLength === "number");
  }

  private static isFull(
    event: LSP.TextDocumentContentChangeEvent,
  ): event is { text: string } {
    const candidate: IncrementalEvent = event as IncrementalEvent;
    return candidate !== undefined && candidate !== null &&
      typeof candidate.text === "string" && candidate.range === undefined &&
      candidate.rangeLength === undefined;
  }
}
