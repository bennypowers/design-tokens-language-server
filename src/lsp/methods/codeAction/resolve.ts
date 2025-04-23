import {
  CodeAction,
  Diagnostic,
  TextEdit,
} from "vscode-languageserver-protocol";

import { DTLSCodeAction } from "../textDocument/codeAction.ts";
import { CssDocument, tsRangeToLspRange } from "#css";
import { DTLSContext } from "#lsp";

import { zip } from "@std/collections/zip";

function fixFallback(action: CodeAction, context: DTLSContext): CodeAction {
  if (
    typeof action.data?.textDocument?.uri === "string" && action.diagnostics
  ) {
    return {
      ...action,
      edit: {
        changes: {
          [action.data.textDocument.uri]: action.diagnostics.map(
            function getEditFromDiagnostic(
              diagnostic: Diagnostic,
            ): TextEdit | undefined {
              const token = context.tokens.get(diagnostic.data.tokenName);
              if (token) {
                // TODO: preserve whitespace
                const newText = token.$value;
                const range = diagnostic.range;
                return { range, newText };
              }
            },
          ).filter((x) => !!x),
        },
      },
    };
  } else {
    return action;
  }
}

function fixAllFallbacks(action: CodeAction, context: DTLSContext): CodeAction {
  if (typeof action.data?.textDocument?.uri === "string") {
    const doc = context.documents.get(action.data.textDocument.uri);
    if (doc.language === "css") {
      const captures = doc.query(CssDocument.queries.VarCallWithFallback);
      const tokenNameCaps = captures.filter((cap) => cap.name === "tokenName");
      const fallbackCaps = captures.filter((cap) => cap.name === "fallback");
      return {
        ...action,
        edit: {
          changes: {
            [action.data.textDocument.uri]: zip(tokenNameCaps, fallbackCaps)
              .flatMap(function ([nameCap, fallbackCap]) {
                const token = context.tokens.get(nameCap.node.text);
                if (token) {
                  const newText = token.$value;
                  const range = tsRangeToLspRange(fallbackCap.node);
                  return [{ range, newText }];
                } else return [];
              }),
          },
        },
      };
    }
  }
  return action;
}

/**
 * Resolves a code action by applying the appropriate fix based on the action's title.
 *
 * @param action - The code action to resolve.
 * @param context - The context containing design tokens and other relevant information.
 * @returns The resolved code action with the appropriate edit applied.
 */
export function resolve(action: CodeAction, context: DTLSContext): CodeAction {
  switch (action.title) {
    case DTLSCodeAction.fixFallback:
      return fixFallback(action, context);
    case DTLSCodeAction.fixAllFallbacks:
      return fixAllFallbacks(action, context);
    default:
      return action;
  }
}
