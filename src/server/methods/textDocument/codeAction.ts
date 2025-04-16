import {
  type CodeAction,
  CodeActionKind,
  type CodeActionParams,
  TextEdit,
} from "vscode-languageserver-protocol";

import { DTLSErrorCodes } from "./diagnostic.ts";
import { tokens } from "../../storage.ts";
import { HardNode } from "https://deno.land/x/deno_tree_sitter@0.2.8.5/tree_sitter.js";
import { captureIsTokenCall, captureIsTokenName, documents, lspRangeIsInTsNode, tsNodeIsInLspRange, tsRangeToLspRange } from "../../css/documents.ts";

export enum DTLSCodeActionTitles {
  fixFallback = "Fix token fallback value",
  fixAllFallbacks = "Fix all token fallback values",
  toggleFallback = "Toggle design token fallback value",
  toggleRangeFallbacks = "Toggle design token fallback values (in range)",
}

function getEditFromTSArgumentsNode(node: HardNode): TextEdit | undefined {
  const [, nameNode, closeParenOrFallback] = node.children;
  const hasFallback = closeParenOrFallback?.text !== ")";
  const token = tokens.get(nameNode.text);
  if (token) {
    // TODO: preserve whitespace
    const newText = hasFallback
      ? `(${nameNode.text})`
      : `(${nameNode.text}, ${token.$value})`;
    const range = tsRangeToLspRange(node);
    return { range, newText };
  }
}

export function codeAction(params: CodeActionParams): null | CodeAction[] {
  const { textDocument } = params;
  const results = documents.queryVarCalls(textDocument.uri);

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
