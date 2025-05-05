import {
  Location,
  ReferenceParams,
  ServerCapabilities,
} from "vscode-languageserver-protocol";
import { DTLSContext } from "#lsp";
import { JsonDocument } from "#json";
import { YamlDocument } from "#yaml";
import { Logger } from "#logger";

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
      // stringify to ensure reference identity and avoid duplicates
      const locations = new Set<string>();
      for (const doc of context.documents.getAll()) {
        if (doc.language === "css") {
          for (const range of doc.getRangesForSubstring(`(${name})`)) {
            locations.add(JSON.stringify({ uri: doc.uri, range }));
          }
          for (const range of doc.getRangesForSubstring(`(${name},`)) {
            locations.add(JSON.stringify({ uri: doc.uri, range }));
          }
        } else {
          for (const range of doc.getRangesForSubstring(ext.reference)) {
            locations.add(JSON.stringify({ uri: doc.uri, range }));
          }
        }
      }
      if (params.context.includeDeclaration) {
        const uri = ext.definitionUri;
        const doc = context.documents.get(uri) as JsonDocument | YamlDocument;
        const range = doc.getRangeForPath(ext.path);
        if (uri && range) {
          locations.add(JSON.stringify({ uri, range }));
        }
      }
      return locations.values().map((x) => JSON.parse(x)).toArray();
    }
  }
}

export const capabilities: Partial<ServerCapabilities> = {
  referencesProvider: true,
};
