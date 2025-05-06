import {
  Hover,
  HoverParams,
  MarkupContent,
  MarkupKind,
  ServerCapabilities,
} from "vscode-languageserver-protocol";

import { getTokenMarkdown } from "#tokens";
import { DTLSContext } from "#lsp";
import { Logger } from "#logger";

export const capabilities: Partial<ServerCapabilities> = {
  hoverProvider: true,
};

/**
 * Generates hover information for design tokens.
 *
 * @param params - The parameters for the hover request.
 * @param context - The context containing design tokens and other information.
 * @returns The hover information containing the token's documentation and range.
 */
export function hover(params: HoverParams, context: DTLSContext): null | Hover {
  const doc = context.documents.get(params.textDocument.uri);
  const result = doc.getTokenReferenceAtPosition(params.position);
  Logger.debug`
uri: ${doc.uri}
result: ${result}
`;
  if (result) {
    const { name, range } = result;
    const token = context.tokens.get(name);
    if (token) {
      const contents: MarkupContent = {
        value: getTokenMarkdown(token),
        kind: MarkupKind.Markdown,
      };
      return { contents, range };
    }
  }
  return null;
}
