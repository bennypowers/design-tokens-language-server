import type { Token } from "style-dictionary";
import {
  CompletionItem,
  CompletionItemKind,
  CompletionParams,
  CompletionList,
  InsertTextFormat,
  InsertTextMode,
  Position,
} from "vscode-languageserver-protocol";

import { all } from "../../storage.ts";

import { getCssSyntaxNodeAtPosition, tsNodeToRange } from "../../css/css.ts";
import { Logger } from "../../logger.ts";

interface NamedToken extends Token {
  name: string;
}

const matchesWord =
(word: string | null) =>
  (x: Token): x is NamedToken =>
    !!word &&
    !!x.name &&
    x.name
        .replaceAll("-", "")
        .startsWith(word.replaceAll("-", ""));

function offset(pos: Position, offset: Partial<Position>): Position {
  return {
    line: pos.line + (offset.line ?? 0),
    character: pos.character + (offset.character ?? 0),
  };
}

export async function completion(params: CompletionParams): Promise<null | CompletionList | CompletionItem[]> {
  await new Promise(r => setTimeout(r));
  const node = getCssSyntaxNodeAtPosition(params.textDocument.uri, offset(params.position, { character: -2 }));
  if (!node) return null;
  // const trigger = params.context?.triggerKind === InlineCompletionTriggerKind.Automatic ?
  //   node.text + params.context.triggerCharacter : node.text;
  try {
    const range = tsNodeToRange(node);
    Logger.debug({ node: node.text, range });
    const items = all().filter(matchesWord(node.text)).map(({ name, $value }) => ({
      label: name,
      kind: 15 satisfies typeof CompletionItemKind.Snippet,
      ...(range ? {
        textEdit: {
          range,
          newText: `var(--${name}\${1|\\, ${$value},|})$0`,
        }
      } : {
        insertText: `var(--${name}\${1|\\, ${$value},|}):0`,
      })
    }) satisfies CompletionItem).toArray();
    return {
      // TODO: perf
      isIncomplete: items.length === 0 || items.length < all().toArray().length,
      itemDefaults: {
        insertTextFormat: InsertTextFormat.Snippet,
        insertTextMode: InsertTextMode.asIs,
        editRange: range,
      },
      items
    }
  } catch (e) {
    Logger.error(`${e}`);
    return null
  }
}
