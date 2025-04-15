import {
  type CodeActionParams,
  type CodeAction,
  CodeActionKind,
  TextEdit,
} from "vscode-languageserver-protocol";
import {
  queryCssDocument,
  tsNodeIsInLspRange,
  tsNodeToLspRange,
  lspRangeIsInTsNode,
  TSQueryCapture,
  SyntaxNode,
} from "../../css/css.ts";
import { tokens } from "../../storage.ts";
import { HardNode } from "https://deno.land/x/deno_tree_sitter@0.2.8.5/tree_sitter.js";

const scheme = String.raw;
const QUERY = scheme`
  (call_expression
    (function_name) @fn
    (arguments) @args
    (#eq? @fn "var")) @call
`;

function captureIsTokenName(cap: TSQueryCapture) {
  return cap.name === 'args' && !!cap.node.children
    .find(child => child.type === 'plain_value'
                && tokens.has(child.text.replace(/^--/, '')));
}

function captureIsTokenCall(cap: TSQueryCapture) {
  return cap.name === 'call' && !!cap.node.children
    .find(child => child.type === 'arguments')
    ?.children
    .some(child => child.type === 'plain_value'
                && tokens.has(child.text.replace(/^--/, '')));
}

function getEdit(node: HardNode): TextEdit | undefined {
  const [, nameNode, closeParenOrFallback] = node.children;
  const hasFallback = closeParenOrFallback.text !== ')';
  const token = tokens.get(nameNode.text);
  if (token) {
    // TODO: preserve whitespace
    const newText = hasFallback ? `(${nameNode.text})` : `(${nameNode.text}, ${token.$value})`;
    const range = tsNodeToLspRange(node as unknown as SyntaxNode);
    return { range, newText }
  }
}

export function codeAction(params: CodeActionParams): null | CodeAction[] {
  const results = queryCssDocument(params.textDocument.uri, QUERY);

  const tokenNameCaptures = results.flatMap(result =>
    result.captures.filter(cap =>
         captureIsTokenName(cap)
      && lspRangeIsInTsNode(cap.node, params.range)));

  const tokenCallCaptures = results.flatMap(result =>
    result.captures.filter(cap =>
         captureIsTokenCall(cap)
      && tsNodeIsInLspRange(cap.node, params.range)));

  const kind = CodeActionKind.RefactorRewrite;
  if (tokenCallCaptures.length) {
    const title = 'Toggle design token fallback values (in range)';
    return [{
      title,
      kind,
      edit: {
        changes: {
          [params.textDocument.uri]: tokenCallCaptures.map(cap => {
            const args = cap.node.children.find(x =>x.type === 'arguments')
            if (args) {
              const edit = getEdit(args);
              if (edit) {
                return edit;
              }
            }
          }).filter(x => !!x),
        },
      },
    }];
  } else if (tokenNameCaptures.length) {
    const [cap] = tokenNameCaptures;
    const title = 'Toggle design token fallback value';
    const edit = getEdit(cap.node)
    if (edit) {
      return [{
        title,
        kind,
        edit: {
          changes: {
            [params.textDocument.uri]: [edit],
          },
        },
      }]
    }
  }
  return null;
}
