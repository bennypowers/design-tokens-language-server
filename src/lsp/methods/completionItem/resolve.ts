import type { CompletionItem, MarkupKind } from "vscode-languageserver-protocol";

import { tokens, getTokenMarkdown } from "#tokens";

/**
 * Resolves a completion item by adding details and documentation.
 *
 * @param params - The completion item to resolve.
 * @returns The resolved completion item with additional details and documentation.
 */
export function resolve(params: CompletionItem): CompletionItem {
  const token = tokens.get(params.label);
  if (!token)
    return params
  else
    return {
      ...params,
      labelDetails: {
        detail: `: ${token.$value}`,
      },
      documentation: token?.$description && {
        value: getTokenMarkdown(params.label, token),
        kind: "markdown" satisfies typeof MarkupKind.Markdown,
      },
    };
}
