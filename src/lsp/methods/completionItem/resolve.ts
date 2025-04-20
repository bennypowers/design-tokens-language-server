import type { CompletionItem, MarkupKind } from "vscode-languageserver-protocol";

import { tokens, getTokenMarkdown } from "#tokens";

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
        value: getTokenMarkdown(`--${params.label}`, token),
        kind: "markdown" satisfies typeof MarkupKind.Markdown,
      },
    };
}
