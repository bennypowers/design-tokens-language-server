import {
  type CodeAction,
  CodeActionKind,
  type CodeActionParams,
  TextEdit,
} from "vscode-languageserver-protocol";

import { DTLSErrorCodes } from "./diagnostic.ts";
import { tokens } from "#tokens";
import type { Node } from "web-tree-sitter";
import {
  captureIsTokenCall,
  captureIsTokenName,
  documents,
  lspRangeIsInTsNode,
  tsNodeIsInLspRange,
  tsRangeToLspRange,
} from "#css";

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

function getEditFromTSArgumentsNode(
  node: Node | null,
): TextEdit | undefined {
  if (node) {
    const [, nameNode, closeParenOrFallback] = node.children;
    const hasFallback = closeParenOrFallback?.text !== ")";
    const token = tokens.get(nameNode?.text!);
    if (token) {
      // TODO: preserve whitespace
      const newText = hasFallback
        ? `(${nameNode?.text})`
        : `(${nameNode?.text}, ${token.$value})`;
      const range = tsRangeToLspRange(node);
      return { range, newText };
    }
  }
}

/**
 * Generates code actions for design tokens.
 *
 * @param params - The parameters for the code action request.
 * @returns An array of code actions representing the fixes or refactorings for design tokens.
 */
export function codeAction(params: CodeActionParams): null | CodeAction[] {
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

  if (diagnostics.length) {
    actions.push({
      title: DTLSCodeAction.fixAllFallbacks,
      kind: CodeActionKind.SourceFixAll,
      data: { textDocument },
    });
  }

  const captures = documents.queryVarCalls(textDocument.uri);

  const tokenNameCaptures = captures.filter((cap) =>
    captureIsTokenName(cap) &&
    lspRangeIsInTsNode(cap.node, params.range)
  );

  const tokenCallCaptures = captures.filter((cap) =>
    captureIsTokenCall(cap) &&
    tsNodeIsInLspRange(cap.node, params.range)
  );

  // TODO: resolve the edits for the tokenCallCaptures

  if (tokenCallCaptures.length) {
    actions.push({
      title: DTLSCodeAction.toggleRangeFallbacks,
      kind: CodeActionKind.RefactorRewrite,
      edit: {
        changes: {
          [textDocument.uri]: tokenCallCaptures.map((cap) => {
            const args = cap.node.children.find((x) => x?.type === "arguments");
            if (args) {
              const edit = getEditFromTSArgumentsNode(args);
              if (edit) {
                return edit;
              }
            }
          }).filter((x) => !!x),
        },
      },
    });
  } else if (tokenNameCaptures.length) {
    const [cap] = tokenNameCaptures;
    const edit = getEditFromTSArgumentsNode(cap.node.parent);
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

  if (actions.length) {
    return actions;
  } else {
    return null;
  }
}
