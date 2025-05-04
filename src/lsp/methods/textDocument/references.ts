import {
  Location,
  ReferenceParams,
  ServerCapabilities,
} from "vscode-languageserver-protocol";
import { DTLSContext } from "#lsp";
import { JsonDocument } from "#json";
import { YamlDocument } from "#yaml";

export function references(
  params: ReferenceParams,
  context: DTLSContext,
): Location[] | null {
  const doc = context.documents.get(params.textDocument.uri);
  if (doc.language === "css") {
    // let css-ls handle it, at least for now
    return null;
  } else if (doc.language === "json" || doc.language === "yaml") {
    const reference = doc.getTokenReferenceAtPosition(params.position);
    if (!reference) return [];
    const { name } = reference;
    const token = context.tokens.get(name);
    if (!token) return [];
    return context.documents.getAll().flatMap((doc) => {
      const { uri } = doc;
      const locations: Location[] = [];
      switch (doc.language) {
        case "css":
          locations.push(
            ...doc
              .getRangesForSubstring(name)
              .map((range) => ({ uri, range })),
          );
          break;
        case "json":
        case "yaml":
          locations.push(
            ...doc
              .getRangesForSubstring(
                token.$extensions.designTokensLanguageServer.reference,
              )
              .map((range) => ({ uri, range })),
          );
          break;
      }
      if (params.context.includeDeclaration) {
        const ext = token.$extensions.designTokensLanguageServer;
        const uri = ext.definitionUri;
        const doc = context.documents.get(uri) as JsonDocument | YamlDocument;
        const range = doc.getRangeForPath(ext.path);
        if (uri && range) {
          locations.push({ uri, range });
        }
      }
      return locations;
    });
  }
  return null;
}

export const capabilities: Partial<ServerCapabilities> = {
  referencesProvider: true,
};
