import type { Token } from "style-dictionary";
import {
  CompletionItem,
  CompletionItemKind,
  CompletionParams,
  CompletionList,
  InlineCompletionTriggerKind,
  InsertTextFormat,
  InsertTextMode,
} from "vscode-languageserver-protocol";

import { all } from "../../storage.ts";

import { getCSSWordAtPosition } from "../../css/css.ts";
import { Logger } from "../../logger.ts";

interface NamedToken extends Token {
  name: string;
}

const good = {
  "label": "additive-symbols",
  "documentation": {
    "kind": "plaintext",
    "value": "@counter-style descriptor. Specifies the symbols used by the marker-construction algorithm specified by the system descriptor. Needs to be specified if the counter system is 'additive'.\n(Firefox 33)\n\nSyntax: [ <integer> && <symbol> ]#"
  },
  "tags": [],
  "textEdit": {
    "range": {
      "start": { "line": 5, "character": 4 },
      "end": { "line": 5, "character": 9 }
    },
    "newText": "additive-symbols: $0;"
  },
  "insertTextFormat": 2,
  "kind": 10,
  "command": {
    "title": "Suggest",
    "command": "editor.action.triggerSuggest"
  },
  "sortText": "d_cd"
}

const matchesWord =
(word: string | null) =>
  (x: Token): x is NamedToken =>
    !!word &&
    !!x.name &&
    x.name
        .replaceAll("-", "")
        .startsWith(word.replaceAll("-", ""));

export async function completion(params: CompletionParams): Promise<null | CompletionList | CompletionItem[]> {
  await new Promise(r => setTimeout(r));
  const { word, range } = getCSSWordAtPosition(params.textDocument.uri, params.position);
  if (!range) return null;
  const trigger = params.context?.triggerKind === InlineCompletionTriggerKind.Automatic ? word + params.context.triggerCharacter : word;
  try {
    const items = all().filter(matchesWord(trigger)).map(({ name, $value }) => ({
      label: name,
      kind: 15 satisfies typeof CompletionItemKind.Snippet,
      textEdit: {
        range,
        newText: `var(--${name}\${1:|\, ${$value},|}):0`,
      }
    }) satisfies CompletionItem).toArray();
    Logger.debug({ word, range });
    return {
      isIncomplete: false,
      itemDefaults: {
        insertTextFormat: 2 satisfies typeof InsertTextFormat.Snippet,
        insertTextMode: 1 satisfies typeof InsertTextMode.asIs,
        editRange: range,
      },
      items
    }
  } catch (e) {
    Logger.error(`${e}`);
    return null
  }
}
