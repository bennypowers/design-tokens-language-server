import * as LSP from "vscode-languageserver-protocol";

import type { DTLSContext } from "#lsp";

import { getTokenMarkdown } from "#tokens";

/**
 * Generates completion items for design tokens.
 *
 * @param params - The parameters for the completion request.
 * @param context - The context containing the design tokens and documents.
 * @returns A completion list or an array of completion items representing the design tokens that match the specified word.
 */
export function completion(
  params: LSP.CompletionParams,
  ctx: DTLSContext,
): null | LSP.CompletionList {
  return ctx
    .documents
    .get(params.textDocument.uri)
    .getCompletions(ctx, params);
}

/**
 * Resolves a completion item by adding details and documentation.
 *
 * @param params - The completion item to resolve.
 * @param context - The context containing design tokens and other information.
 * @returns The resolved completion item with additional details and documentation.
 */
export function resolve(
  params: LSP.CompletionItem,
  context: DTLSContext,
): LSP.CompletionItem {
  const tokenName = params.data?.tokenName ?? params.label;
  const token = context.tokens.get(tokenName);
  if (!token) {
    return params;
  } else {
    return {
      ...params,
      labelDetails: {
        detail: `: ${token.$value}`,
      },
      documentation: {
        value: getTokenMarkdown(token),
        kind: "markdown" satisfies typeof LSP.MarkupKind.Markdown,
      },
    };
  }
}

export const capabilities: Partial<LSP.ServerCapabilities> = {
  completionProvider: {
    // triggerCharacters: ["{", "-", '"', "'"],
    resolveProvider: true,
    completionItem: {
      labelDetailsSupport: true,
    },
  },
};
