import type { CompletionItem, MarkupKind } from "vscode-languageserver-protocol";

import { get } from "../../storage.ts";

import { getTokenMarkdown } from "../../markdown.ts";

export function resolve(params: CompletionItem): CompletionItem {
  const token = get(params.label);
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
