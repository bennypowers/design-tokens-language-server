import {
  type CodeAction,
  CodeActionKind,
  type CodeActionParams,
  TextEdit,
} from "vscode-languageserver-protocol";
import {
  captureIsTokenCall,
  captureIsTokenName,
  lspRangeIsInTsNode,
  queryCssDocument,
  type SyntaxNode,
  tsNodeIsInLspRange,
  tsNodeToLspRange,
} from "../../tree-sitter/css.ts";

import { VarCall } from "../../tree-sitter/css/queries.ts";
import { DTLSErrorCodes } from "./diagnostic.ts";
import { tokens } from "../../storage.ts";
import { HardNode } from "https://deno.land/x/deno_tree_sitter@0.2.8.5/tree_sitter.js";

export enum DTLSCodeActionTitles {
  fixFallback = "Fix token fallback value",
  fixAllFallbacks = "Fix all token fallback values",
  toggleFallback = "Toggle design token fallback value",
  toggleRangeFallbacks = "Toggle design token fallback values (in range)",
}

function getEditFromTSNode(node: HardNode): TextEdit | undefined {
  const [, nameNode, closeParenOrFallback] = node.children;
  const hasFallback = closeParenOrFallback.text !== ")";
  const token = tokens.get(nameNode.text);
  if (token) {
    // TODO: preserve whitespace
    const newText = hasFallback
      ? `(${nameNode.text})`
      : `(${nameNode.text}, ${token.$value})`;
    const range = tsNodeToLspRange(node as unknown as SyntaxNode);
    return { range, newText };
  }
}

export function codeAction(params: CodeActionParams): null | CodeAction[] {
  const { textDocument } = params;
  const results = queryCssDocument(textDocument.uri, VarCall);

  const diagnostics = params.context.diagnostics.filter((d) =>
    d.code === DTLSErrorCodes.incorrectFallback
  );

  const actions = [];

  const fixes: CodeAction[] = diagnostics
    .map((d) => ({
      title: DTLSCodeActionTitles.fixFallback,
      kind: CodeActionKind.QuickFix,
      data: { textDocument },
      diagnostics: [d],
    }));

  actions.push(...fixes);

  if (diagnostics.length) {
    actions.push({
      title: DTLSCodeActionTitles.fixAllFallbacks,
      kind: CodeActionKind.SourceFixAll,
      data: { textDocument },
    });
  }

  const tokenNameCaptures = results.flatMap((result) =>
    result.captures.filter((cap) =>
      captureIsTokenName(cap) &&
      lspRangeIsInTsNode(cap.node, params.range)
    )
  );

  const tokenCallCaptures = results.flatMap((result) =>
    result.captures.filter((cap) =>
      captureIsTokenCall(cap) &&
      tsNodeIsInLspRange(cap.node, params.range)
    )
  );

  if (tokenCallCaptures.length) {
    actions.push({
      title: DTLSCodeActionTitles.toggleRangeFallbacks,
      kind: CodeActionKind.RefactorRewrite,
      edit: {
        changes: {
          [textDocument.uri]: tokenCallCaptures.map((cap) => {
            const args = cap.node.children.find((x) => x.type === "arguments");
            if (args) {
              const edit = getEditFromTSNode(args);
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
    const edit = getEditFromTSNode(cap.node);
    if (edit) {
      actions.push({
        title: DTLSCodeActionTitles.toggleFallback,
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
