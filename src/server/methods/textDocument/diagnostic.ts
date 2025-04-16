import { DiagnosticSeverity, DocumentDiagnosticReportKind, type DocumentDiagnosticParams, type DocumentDiagnosticReport } from "vscode-languageserver-protocol";

import { queryCssDocument, tsNodeToLspRange } from "../../tree-sitter/css.ts";
import { VarCallWithFallback } from "../../tree-sitter/css/queries.ts";
import { tokens } from "../../storage.ts";

export enum DTLSErrorCodes {
  incorrectFallback = 'incorrect-fallback',
}

export function diagnostic(params: DocumentDiagnosticParams): DocumentDiagnosticReport {
  const results = queryCssDocument(params.textDocument.uri, VarCallWithFallback);
  return {
    kind: DocumentDiagnosticReportKind.Full,
    items: results.flatMap(result => {
      const tokenNameCap = result.captures.find(x => x.name === 'tokenName');
      const fallbackCap = result.captures.find(x => x.name === 'fallback');
      if (tokenNameCap && fallbackCap && tokens.has(tokenNameCap.node.text)) {
        const tokenName = tokenNameCap.node.text;
        const fallback = fallbackCap.node.text;
        const token = tokens.get(tokenName)!;
        const valid = fallback === token.$value;
        if (!valid)
          return [{
            range: tsNodeToLspRange(fallbackCap.node),
            severity: DiagnosticSeverity.Error,
            message: `Token fallback does not match expected value: ${token.$value}`,
            code: DTLSErrorCodes.incorrectFallback,
            data: {
              tokenName
            }
          }]
      }
      return []
    })
  };
}
