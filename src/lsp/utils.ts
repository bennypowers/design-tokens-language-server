import * as LSP from "vscode-languageserver-protocol";

/**
 * Does the first range contain the second range?
 *
 * @param range - The supposedly outer range.
 * @param otherRange - The supposedly inner range.
 * @return whether the first range contains the second range.
 */
export function lspRangeContains(
  range: LSP.Range,
  otherRange: LSP.Range,
): boolean {
  return range.start.line <= otherRange.start.line &&
    range.end.line >= otherRange.end.line &&
    range.start.character <= otherRange.start.character &&
    range.end.character >= otherRange.end.character;
}
