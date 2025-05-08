import * as LSP from "vscode-languageserver-protocol";

import { TokenVarCall } from "#css";

import { DTLSContext, DTLSErrorCodes } from "#lsp";

import { lspRangeContains } from "#lsp/utils.ts";

function rangeIsSingleChar(range: LSP.Range): boolean {
  return (
    range.start.line === range.end.line &&
    range.start.character === range.end.character
  );
}

function fixFallback(
  action: LSP.CodeAction,
  context: DTLSContext,
): LSP.CodeAction {
  if (
    typeof action.data?.textDocument?.uri === "string" && action.diagnostics
  ) {
    return {
      ...action,
      edit: {
        changes: {
          [action.data.textDocument.uri]: action.diagnostics.map(
            function getEditFromDiagnostic(
              diagnostic: LSP.Diagnostic,
            ): LSP.TextEdit | undefined {
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

function hasInvalidFallback(
  call: TokenVarCall,
): call is TokenVarCall & { fallback: { valid: false } } {
  return !!call.fallback && !call.fallback.valid;
}

function fixAllFallbacks(
  action: LSP.CodeAction,
  context: DTLSContext,
): LSP.CodeAction {
  if (typeof action.data?.textDocument?.uri === "string") {
    const doc = context.documents.get(action.data.textDocument.uri);
    if (doc.language === "css") {
      return {
        ...action,
        edit: {
          changes: {
            [action.data.textDocument.uri]: doc
              .varCalls
              .filter(hasInvalidFallback)
              .map((call) => ({
                range: call.fallback.range,
                newText: call.token.token.$value,
              })),
          },
        },
      };
    }
  }
  return action;
}

function getEditFromTokenVarCall(
  call: TokenVarCall,
  context: DTLSContext,
): LSP.TextEdit | undefined {
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

/**
 * Generates code actions for design tokens.
 *
 * @param params - The parameters for the code action request.
 * @param context - The context containing the design tokens and documents.
 * @returns An array of code actions representing the fixes or refactorings for design tokens.
 */
export function codeAction(
  params: LSP.CodeActionParams,
  context: DTLSContext,
): null | LSP.CodeAction[] {
  const { textDocument } = params;

  const diagnostics = params.context.diagnostics.filter((d) =>
    d.code === DTLSErrorCodes.incorrectFallback
  );

  const actions = [];

  const fixes: LSP.CodeAction[] = diagnostics
    .map((d) => ({
      title: DTLSCodeAction.fixFallback,
      kind: LSP.CodeActionKind.QuickFix,
      data: { textDocument },
      diagnostics: [d],
    }));

  actions.push(...fixes);

  if (diagnostics.length > 1) {
    actions.push({
      title: DTLSCodeAction.fixAllFallbacks,
      kind: LSP.CodeActionKind.SourceFixAll,
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
          kind: LSP.CodeActionKind.RefactorRewrite,
          edit: {
            changes: {
              [textDocument.uri]: doc.varCalls.map((call) =>
                getEditFromTokenVarCall(call, context)
              )
                .filter((x) => !!x),
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
            kind: LSP.CodeActionKind.RefactorRewrite,
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

/**
 * Resolves a code action by applying the appropriate fix based on the action's title.
 *
 * @param action - The code action to resolve.
 * @param context - The context containing design tokens and other relevant information.
 * @returns The resolved code action with the appropriate edit applied.
 */
export function resolve(
  action: LSP.CodeAction,
  context: DTLSContext,
): LSP.CodeAction {
  switch (action.title) {
    case DTLSCodeAction.fixFallback:
      return fixFallback(action, context);
    case DTLSCodeAction.fixAllFallbacks:
      return fixAllFallbacks(action, context);
    default:
      return action;
  }
}

export const capabilities: Partial<LSP.InitializeResult> = {
  codeActionProvider: {
    resolveProvider: true,
    codeActionKinds: [
      LSP.CodeActionKind.QuickFix,
      LSP.CodeActionKind.RefactorRewrite,
      LSP.CodeActionKind.SourceFixAll,
    ],
  },
};
