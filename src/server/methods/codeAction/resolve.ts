import {
  CodeAction,
  Diagnostic,
  TextEdit,
} from "vscode-languageserver-protocol";
import { tokens } from "../../storage.ts";
import { DTLSCodeActionTitles } from "../textDocument/codeAction.ts";
import { VarCallWithFallback } from "../../tree-sitter/css/queries.ts";
import {
  queryCssDocument,
  tsNodeToLspRange,
  TSQueryCapture,
} from "../../tree-sitter/css.ts";

function getEditFromDiagnostic(diagnostic: Diagnostic): TextEdit | undefined {
  const token = tokens.get(diagnostic.data.tokenName);
  if (token) {
    // TODO: preserve whitespace
    const newText = token.$value;
    const range = diagnostic.range;
    return { range, newText };
  }
}

function getEditFromVarFallbackCap(cap: TSQueryCapture): TextEdit | undefined {
  const token = tokens.get(cap.node.text);
  if (token) {
    const newText = token.$value;
    const range = tsNodeToLspRange(cap.node);
    return { range, newText };
  }
}

function fixFallback(action: CodeAction): CodeAction {
  if (
    typeof action.data?.textDocument?.uri === "string" && action.diagnostics
  ) {
    return {
      ...action,
      edit: {
        changes: {
          [action.data.textDocument.uri]: action.diagnostics.map(
            getEditFromDiagnostic,
          ).filter((x) => !!x),
        },
      },
    };
  } else {
    return action;
  }
}

function fixAllFallbacks(action: CodeAction): CodeAction {
  if (typeof action.data?.textDocument?.uri === "string") {
    const results = queryCssDocument(
      action.data.textDocument.uri,
      VarCallWithFallback,
    );
    return {
      ...action,
      edit: {
        changes: {
          [action.data.textDocument.uri]: results.flatMap((r) =>
            r.captures.map(getEditFromVarFallbackCap)
          ).filter((x) => !!x),
        },
      },
    };
  } else {
    return action;
  }
}

export function resolve(action: CodeAction): CodeAction {
  switch (action.title) {
    case DTLSCodeActionTitles.fixFallback:
      return fixFallback(action);
    case DTLSCodeActionTitles.fixAllFallbacks:
      return fixAllFallbacks(action);
    default:
      return action;
  }
}
