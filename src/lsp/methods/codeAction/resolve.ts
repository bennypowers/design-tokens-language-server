import {
  CodeAction,
  Diagnostic,
  TextEdit,
} from "vscode-languageserver-protocol";
import { tokens } from "#tokens";
import { DTLSCodeAction } from "../textDocument/codeAction.ts";
import { documents, tsRangeToLspRange } from "#css";
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

/**
 * Resolves a code action by applying the appropriate fix based on the action's title.
 *
 * @param action - The code action to resolve.
 * @returns The resolved code action with the appropriate edit applied.
 */
export function resolve(action: CodeAction): CodeAction {
  switch (action.title) {
    case DTLSCodeAction.fixFallback:
      return fixFallback(action);
    case DTLSCodeAction.fixAllFallbacks:
      return fixAllFallbacks(action);
    default:
      return action;
  }
}
