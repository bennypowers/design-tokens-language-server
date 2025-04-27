import {
  type CodeAction,
  CodeActionKind,
  type CodeActionParams,
  Range,
  TextEdit,
} from "vscode-languageserver-protocol";

import { TokenVarCall } from "#css";

import { DTLSContext, DTLSErrorCodes } from "#lsp";
import { lspRangeContains } from "#lsp/utils.ts";

function rangeIsSingleChar(range: Range): boolean {
  return (
    range.start.line === range.end.line &&
    range.start.character === range.end.character
  );
}

export enum DTLSCodeAction {
  /** Fix the fallback value of a design token.*/
  fixFallback = "Fix token fallback value",
  /** Fix all fallback values of design tokens. */
  fixAllFallbacks = "Fix all token fallback values",
  /** Toggle the fallback value of a design token.* */
  toggleFallback = "Toggle design token fallback value",
  /** Toggle the fallback value of a design token in a range. */
  toggleRangeFallbacks = "Toggle design token fallback values (in range)",
}

function getEditFromTokenVarCall(
  call: TokenVarCall,
  context: DTLSContext,
): TextEdit | undefined {
  if (call) {
    const { range, token, fallback } = call;
    const { $value } = context.tokens.get(token.name)!;
    if (token) {
      const hasFallback = !!fallback;
      // TODO: preserve whitespace
      const newText = hasFallback
        ? `var(${token.name})`
        : `var(${token.name}, ${$value})`;
      return { range, newText };
    }
  }
}

/**
 * Generates code actions for design tokens.
 *
 * @param params - The parameters for the code action request.
 * @param context - The context containing the design tokens and documents.
 * @returns An array of code actions representing the fixes or refactorings for design tokens.
 */
export function codeAction(
  params: CodeActionParams,
  context: DTLSContext,
): null | CodeAction[] {
  const { textDocument } = params;

  const diagnostics = params.context.diagnostics.filter((d) =>
    d.code === DTLSErrorCodes.incorrectFallback
  );

  const actions = [];

  const fixes: CodeAction[] = diagnostics
    .map((d) => ({
      title: DTLSCodeAction.fixFallback,
      kind: CodeActionKind.QuickFix,
      data: { textDocument },
      diagnostics: [d],
    }));

  actions.push(...fixes);

  if (diagnostics.length > 1) {
    actions.push({
      title: DTLSCodeAction.fixAllFallbacks,
      kind: CodeActionKind.SourceFixAll,
      data: { textDocument },
    });
  }

  const doc = context.documents.get(textDocument.uri);

  if (doc.language === "css") {
    // TODO: resolve the edits for the tokenCallCaptures
    if (!rangeIsSingleChar(params.range)) {
      if (doc.varCalls.length) {
        actions.push({
          title: DTLSCodeAction.toggleRangeFallbacks,
          kind: CodeActionKind.RefactorRewrite,
          edit: {
            changes: {
              [textDocument.uri]: doc.varCalls.map((call) =>
                getEditFromTokenVarCall(call, context)
              ).filter((x) => !!x),
            },
          },
        });
      }
    } else {
      const call = doc.varCalls.find((call) =>
        lspRangeContains(call.range, params.range)
      );
      if (call) {
        const edit = getEditFromTokenVarCall(call, context);
        if (edit) {
          actions.push({
            title: DTLSCodeAction.toggleFallback,
            kind: CodeActionKind.RefactorRewrite,
            edit: {
              changes: {
                [textDocument.uri]: [edit],
              },
            },
          });
        }
      }
    }

    if (actions.length) {
      return actions;
    }
  }

  return null;
}
