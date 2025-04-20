import { Documents } from "#css";
import { TokenMap } from "#tokens";

export class TestDocuments extends Documents {
  create(text: string, tokens: TokenMap) {
    const id = this.allDocuments.length;
    const uri = `file:///test-${id}.css`;
    this.onDidOpen({
      textDocument: { uri, languageId: "css", version: 1, text },
    }, tokens);
    return uri;
  }

  tearDown() {
    for (const doc of this.allDocuments) {
      this.onDidClose({ textDocument: { uri: doc.uri } });
    }
  }
}
