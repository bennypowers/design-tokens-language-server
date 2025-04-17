import {
  CodeAction,
  Diagnostic,
  TextEdit,
} from "vscode-languageserver-protocol";
import { tokens } from "../../storage.ts";
import { DTLSCodeActionTitles } from "../textDocument/codeAction.ts";
import { documents, tsRangeToLspRange } from "../../css/documents.ts";
import type { QueryCapture } from 'web-tree-sitter';
import { Logger } from "../../logger.ts";
import { zip } from "@std/collections/zip";

function getEditFromDiagnostic(diagnostic: Diagnostic): TextEdit | undefined {
  const token = tokens.get(diagnostic.data.tokenName);
  if (token) {
    // TODO: preserve whitespace
    const newText = token.$value;
    const range = diagnostic.range;
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
    const captures = documents.queryVarCallsWithFallback(action.data.textDocument.uri);
    const tokenNameCaps = captures.filter(cap => cap.name === 'tokenName');
    const fallbackCaps = captures.filter(cap => cap.name === 'fallback');
    return {
      ...action,
      edit: {
        changes: {
          [action.data.textDocument.uri]: zip(tokenNameCaps, fallbackCaps)
            .flatMap(function([nameCap, fallbackCap]) {
              const token = tokens.get(nameCap.node.text);
              if (token) {
                const newText = token.$value;
                const range = tsRangeToLspRange(fallbackCap.node);
                return [{ range, newText }];
              } else return [];
            })
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
