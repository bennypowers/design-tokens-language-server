import { CodeAction, Diagnostic, TextEdit } from 'vscode-languageserver-protocol';

import { DTLSCodeAction } from '../textDocument/codeAction.ts';
import { TokenVarCall } from '#css';
import { DTLSContext } from '#lsp';

function fixFallback(action: CodeAction, context: DTLSContext): CodeAction {
  if (
    typeof action.data?.textDocument?.uri === 'string' && action.diagnostics
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

function hasInvalidFallback(
  call: TokenVarCall,
): call is TokenVarCall & { fallback: { valid: false } } {
  return !!call.fallback && !call.fallback.valid;
}

function fixAllFallbacks(action: CodeAction, context: DTLSContext): CodeAction {
  if (typeof action.data?.textDocument?.uri === 'string') {
    const doc = context.documents.get(action.data.textDocument.uri);
    if (doc.language === 'css') {
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
