import * as LSP from "vscode-languageserver-protocol";
import { findNodeAtLocation, type Node, parseTree } from "npm:jsonc-parser";
import { DTLSContext } from "#lsp";

function sonNodeToLspRange(node: Node, content: string): LSP.Range {
  const start = node.offset;
  const end = start + node.length;

  return {
    start: offsetToPosition(content, start),
    end: offsetToPosition(content, end),
  };
}

function offsetToPosition(content: string, offset: number): LSP.Position {
  const lines = content.split("\n");
  let line = 0;
  let column = offset;

  for (let i = 0; i < lines.length; i++) {
    if (column <= lines[i].length) {
      line = i;
      break;
    }
    column -= lines[i].length + 1; // +1 for the newline character
  }

  return { line, character: column };
}

export async function definition(
  params: LSP.DefinitionParams,
  context: DTLSContext,
): Promise<LSP.Location[]> {
  const node = context.documents.getNodeAtPosition(
    params.textDocument.uri,
    params.position,
  );

  if (node) {
    const token = context.tokens.get(node.text);
    if (token) {
      const spec = context.tokens.meta.get(token);
      if (spec?.path) {
        const tokenPath = node.text.replace(/^--/, "")
          .split("-")
          .filter((x) => !!x)
          .filter((x) => spec.prefix ? x !== spec.prefix : true);

        const url = new URL(spec.path, params.textDocument.uri);
        const fileContent = await Deno.readTextFile(url);

        const parsedNodes = parseTree(fileContent);
        if (parsedNodes) {
          const node = findNodeAtLocation(parsedNodes, tokenPath);
          if (node) {
            return [{
              uri: url.href,
              range: sonNodeToLspRange(node, fileContent),
            }];
          }
        }
      }
    }
  }
  return [];
}
