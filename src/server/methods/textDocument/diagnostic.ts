import { DocumentDiagnosticReportKind, DocumentDiagnosticParams, DocumentDiagnosticReport } from "vscode-languageserver-protocol";

import { documents } from "../../css/documents.ts";

export enum DTLSErrorCodes {
  incorrectFallback = 'incorrect-fallback',
}

export function diagnostic(params: DocumentDiagnosticParams): DocumentDiagnosticReport {
  return {
    kind: DocumentDiagnosticReportKind.Full,
    items: documents.getDiagnostics(params.textDocument.uri),
  };
}
