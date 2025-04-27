import type {
  ColorInformation,
  DocumentColorParams,
} from "vscode-languageserver-protocol";

import { DTLSContext } from "#lsp";
import { Logger } from "#logger";

/**
 * Generates color information for design tokens.
 *
 * @param params - The parameters for the document color request.
 * @param context - The context containing design tokens and documents.
 * @returns An array of color information representing the design tokens found in the specified document.
 */
export function documentColor(
  params: DocumentColorParams,
  context: DTLSContext,
): ColorInformation[] {
  const doc = context.documents.get(params.textDocument.uri);
  Logger.debug`json document color: ${doc.colors}`;
  return doc.colors;
}
