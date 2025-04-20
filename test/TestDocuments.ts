import { Documents } from "#css";

export class TestDocuments extends Documents {
  create(text: string) {
    const id = this.allDocuments.length;
    const uri = `file:///test-${id}.css`;
    this.onDidOpen({
      textDocument: { uri, languageId: "css", version: 1, text },
    });
    return uri;
  }

  tearDown() {
    for (const doc of this.allDocuments) {
      this.onDidClose({ textDocument: { uri: doc.uri } });
    }
  }
}
