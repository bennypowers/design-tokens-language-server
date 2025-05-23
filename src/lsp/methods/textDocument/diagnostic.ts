import {
  DocumentDiagnosticParams,
  DocumentDiagnosticReportKind,
  RelatedFullDocumentDiagnosticReport,
  ServerCapabilities,
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
  context: DTLSContext,
): RelatedFullDocumentDiagnosticReport {
  return {
    kind: DocumentDiagnosticReportKind.Full,
    items: context
      .documents
      .get(params.textDocument.uri)
      .getDiagnostics(context),
  };
}

export const capabilities: Partial<ServerCapabilities> = {
  diagnosticProvider: {
    interFileDependencies: false,
    workspaceDiagnostics: false,
  },
};
