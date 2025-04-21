import { Documents } from "#css";
import { TokenMap } from "#tokens";

export class TestDocuments extends Documents {
  #tokens: TokenMap;
  constructor(tokens: TokenMap) {
    super();
    this.#tokens = tokens;
  }
  create(text: string) {
    const id = this.allDocuments.length;
    const uri = `file:///test-${id}.css`;
    this.onDidOpen({
      textDocument: { uri, languageId: "css", version: 1, text },
    }, this.#tokens);
    return uri;
  }

  tearDown() {
    for (const doc of this.allDocuments) {
      this.onDidClose({ textDocument: { uri: doc.uri } });
    }
  }
}
