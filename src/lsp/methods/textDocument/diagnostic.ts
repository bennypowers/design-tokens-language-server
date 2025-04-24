import {
  DocumentDiagnosticParams,
  DocumentDiagnosticReportKind,
  RelatedFullDocumentDiagnosticReport,
} from "vscode-languageserver-protocol";

import { DTLSContext } from "#lsp";

/**
 * Generates a full document diagnostic report.
 *
 * @param params - The parameters for the document diagnostic request.
 * @param context - The context containing the design tokens and documents.
 * @returns A full document diagnostic report containing the diagnostics for the specified document.
 */
export function diagnostic(
  params: DocumentDiagnosticParams,
  { documents }: DTLSContext,
): RelatedFullDocumentDiagnosticReport {
  return {
    kind: DocumentDiagnosticReportKind.Full,
    items: documents.get(params.textDocument.uri).diagnostics,
  };
}
