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
      const token = context.tokens.get(reference?.name);
      if (!token) return null;
      const ext = token.$extensions.designTokensLanguageServer;
      // stringify to ensure reference identity and avoid duplicates
      const locations = new Set<string>();
      for (const doc of context.documents.getAll()) {
        if (doc.language === "css") {
          for (const range of doc.getRangesForSubstring(ext.name)) {
            const charAfterRef = doc.getText({
              start: range.end,
              end: {
                line: range.end.line,
                character: range.end.character + 1,
              },
            });
            // it's a var call with or without fallback
            // this excludes things like `--token-color-red:` and `--token-color-reddish)`
            if (charAfterRef.match(/[,)]/)) {
              locations.add(JSON.stringify({ uri: doc.uri, range }));
            }
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
