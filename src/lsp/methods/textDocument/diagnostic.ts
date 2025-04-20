import { DocumentDiagnosticReportKind, DocumentDiagnosticParams, DocumentDiagnosticReport } from "vscode-languageserver-protocol";

import { documents } from "#css";

export enum DTLSErrorCodes {
  /** The fallback value of a design token is incorrect. */
  incorrectFallback = 'incorrect-fallback',
}

/**
 * Generates a full document diagnostic report.
 *
 * @param params - The parameters for the document diagnostic request.
 * @returns A full document diagnostic report containing the diagnostics for the specified document.
 */
export function diagnostic(params: DocumentDiagnosticParams): DocumentDiagnosticReport {
  return {
    kind: DocumentDiagnosticReportKind.Full,
    items: documents.getDiagnostics(params.textDocument.uri),
  };
}
