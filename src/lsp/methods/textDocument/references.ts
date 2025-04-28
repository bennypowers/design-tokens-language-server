import {
  Location,
  ReferenceParams,
  ServerCapabilities,
} from "vscode-languageserver-protocol";
import { DTLSContext } from "#lsp/lsp.ts";

export function references(
  params: ReferenceParams,
  context: DTLSContext,
): Location[] | null {
  const doc = context.documents.get(params.textDocument.uri);
  if (doc.language === "css") {
    // let css-ls handle it, at least for now
    return null;
  } else if (doc.language === "json") {
    const hover = doc.getHoverTokenAtPosition(params.position);
    if (!hover?.path) return [];
    const { name, path } = hover;
    return context.documents.getAll("json").flatMap((jsonDoc) => {
      const { uri } = jsonDoc;
      const refs: Location[] = jsonDoc
        .getRangesForSubstring(name)
        .map((range) => ({ uri, range }));
      if (params.context.includeDeclaration) {
        const range = jsonDoc.getRangeForPath(path);
        if (uri && range) {
          refs.push({ uri, range });
        }
      }
      return refs;
    });
  }
  return null;
}

export const capabilities: Partial<ServerCapabilities> = {
  referencesProvider: true,
};
