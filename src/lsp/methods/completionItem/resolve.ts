import type {
  CompletionItem,
  MarkupKind,
} from "vscode-languageserver-protocol";

import { getTokenMarkdown } from "#tokens";

import { DTLSContext } from "#lsp";

/**
 * Resolves a completion item by adding details and documentation.
 *
 * @param params - The completion item to resolve.
 * @param context - The context containing design tokens and other information.
 * @returns The resolved completion item with additional details and documentation.
 */
export function resolve(
  params: CompletionItem,
  context: DTLSContext,
): CompletionItem {
  const token = context.tokens.get(params.label);
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
        kind: "markdown" satisfies typeof MarkupKind.Markdown,
      },
    };
  }
}
