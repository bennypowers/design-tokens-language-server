import type {
  CompletionItem,
  MarkupKind,
  InsertTextMode,
  InsertTextFormat,
  CompletionItemKind,
} from "vscode-languageserver-protocol";


import { get } from "../../storage.ts";

import { getTokenMarkdown } from "../../token.ts";

export function resolve(params: CompletionItem): CompletionItem {
  const token = get(params.label);
  if (!token)
    return params
  else
    return {
      ...params,
      kind: 15 satisfies typeof CompletionItemKind.Snippet,
      insertText: `var(--${token.name})`,
      insertTextFormat: 2 satisfies typeof InsertTextFormat.Snippet,
      insertTextMode: 1 satisfies typeof InsertTextMode.asIs,
      documentation: token?.$description && {
        value: getTokenMarkdown(token),
        kind: "markdown" satisfies typeof MarkupKind.Markdown,
      },
    };
}
