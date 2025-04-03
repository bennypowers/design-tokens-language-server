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
      documentation: token?.$description && {
        value: getTokenMarkdown(token),
        kind: "markdown" satisfies typeof MarkupKind.Markdown,
      },
    };
}
