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
  switch (doc.language) {
    case "css":
      // let css-ls handle it, at least for now
      return null;
    default: {
      const reference = doc.getTokenReferenceAtPosition(params.position);
      if (!reference) return null;
      const { name } = reference;
      const token = context.tokens.get(name);
      if (!token) return null;
      const ext = token.$extensions.designTokensLanguageServer;
      const locations = new Set<Location>();
      for (const doc of context.documents.getAll()) {
        for (const range of doc.getRangesForSubstring(ext.reference)) {
          locations.add({ uri: doc.uri, range });
        }
      }
      if (params.context.includeDeclaration) {
        const uri = ext.definitionUri;
        const doc = context.documents.get(uri) as JsonDocument | YamlDocument;
        const range = doc.getRangeForPath(ext.path);
        if (uri && range) {
          locations.add({ uri, range });
        }
      }
      return locations.values().toArray();
    }
  }
}

export const capabilities: Partial<ServerCapabilities> = {
  referencesProvider: true,
};
