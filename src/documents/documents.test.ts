import { beforeEach, describe, it } from "jsr:@std/testing/bdd";
import { AnyConstructor, expect } from "jsr:@std/expect";

import { Documents } from "#documents";
import { CssDocument } from "#css";
import { JsonDocument } from "#json";

import { createTestContext, DTLSTestContext } from "#test-helpers";
import { YamlDocument } from "#yaml";

/** a comprehensive test suite for the Documents class */
describe("Documents", () => {
  let ctx: DTLSTestContext;
  let documents: Documents;
  let onDidChange: Documents["handlers"]["textDocument/didChange"];
  let onDidOpen: Documents["handlers"]["textDocument/didOpen"];
  let onDidClose: Documents["handlers"]["textDocument/didClose"];

  beforeEach(async () => {
    ctx = await createTestContext({ testTokensSpecs: [] });
    documents = new Documents();
    onDidOpen = documents.handlers["textDocument/didOpen"];
    onDidChange = documents.handlers["textDocument/didChange"];
    onDidClose = documents.handlers["textDocument/didClose"];
  });

  it("should have handlers for didOpen, didChange, and didClose", () => {
    expect(documents.handlers).toHaveProperty("textDocument/didOpen");
    expect(documents.handlers).toHaveProperty("textDocument/didChange");
    expect(documents.handlers).toHaveProperty("textDocument/didClose");
  });

  describe("getting an unknown document", () => {
    it("should throw an error", () => {
      expect(() => documents.get("file:///unknown.json")).toThrow(
        "ENOENT: no Document found for file:///unknown.json",
      );
    });
  });

  describe("for an unknown language", () => {
    describe("textDocument/didOpen", () => {
      it("should throw", () => {
        expect(() =>
          onDidOpen({
            textDocument: {
              uri: "file:///test.txt",
              languageId: "txt",
              version: 1,
              text: "unsupported language",
            },
          }, ctx)
        )
          .toThrow(
            "Unsupported language: txt",
          );
      });
    });
  });

  describe("for a css file", () => {
    const uri = "file:///test.css";
    const languageId = "css";
    const initialText = "body { color: red; }";

    beforeEach(() => {
      onDidOpen({
        textDocument: { uri, languageId, version: 1, text: initialText },
      }, ctx);
    });

    it("should add a CssDocument to the documents map", () => {
      const doc = documents.get(uri);
      expect(doc).toBeInstanceOf(CssDocument as unknown as AnyConstructor);
      expect(doc.uri).toEqual(uri);
      expect(doc.version).toEqual(1);
      expect(doc.languageId).toEqual(languageId);
      expect(doc.getText()).toEqual(initialText);
    });

    describe("textDocument/didChange", () => {
      it("should update the document", () => {
        onDidChange({
          textDocument: {
            uri,
            version: 2,
          },
          contentChanges: [
            {
              text: "body { color: blue; }",
            },
          ],
        }, ctx);
        const doc = documents.get(uri);
        expect(doc.version).toEqual(2);
        expect(doc.getText()).toEqual("body { color: blue; }");
      });
    });

    describe("textDocument/didClose", () => {
      beforeEach(() => {
        onDidClose({
          textDocument: { uri },
        }, ctx);
      });

      it("should remove the document from the map", () => {
        expect(() => documents.get(uri)).toThrow(
          `ENOENT: no Document found for ${uri}`,
        );
      });
    });
  });

  describe("for a json file", () => {
    const uri = "file:///test.json";
    const languageId = "json";
    const initialText = '{"key": "value"}';

    describe("textDocument/didOpen", () => {
      beforeEach(() => {
        onDidOpen({
          textDocument: { uri, languageId, version: 1, text: initialText },
        }, ctx);
      });

      it("should add a JsonDocument to the documents map", () => {
        const doc = documents.get(uri);
        expect(doc).toBeInstanceOf(JsonDocument as unknown as AnyConstructor);
        expect(doc.uri).toEqual(uri);
        expect(doc.version).toEqual(1);
        expect(doc.languageId).toEqual(languageId);
        expect(doc.getText()).toEqual(initialText);
      });

      describe("textDocument/didChange", () => {
        it("should update the document", () => {
          onDidChange({
            textDocument: {
              uri,
              version: 2,
            },
            contentChanges: [
              {
                text: '{"key": "new value"}',
              },
            ],
          }, ctx);
          const doc = documents.get(uri);
          expect(doc.version).toEqual(2);
          expect(doc.getText()).toEqual('{"key": "new value"}');
        });
      });

      describe("textDocument/didClose", () => {
        beforeEach(() => {
          onDidClose({ textDocument: { uri } }, ctx);
        });

        it("should remove the document from the map", () => {
          expect(() => documents.get(uri)).toThrow(
            `ENOENT: no Document found for ${uri}`,
          );
        });
      });
    });
  });

  describe("for a yaml file", () => {
    const uri = "file:///test.yaml";
    const languageId = "yaml";
    const initialText = 'key: "value"';

    describe("textDocument/didOpen", () => {
      beforeEach(() => {
        onDidOpen({
          textDocument: { uri, languageId, version: 1, text: initialText },
        }, ctx);
      });

      it("should add a YamlDocument to the documents map", () => {
        const doc = documents.get(uri);
        expect(doc).toBeInstanceOf(YamlDocument as unknown as AnyConstructor);
        expect(doc.uri).toEqual(uri);
        expect(doc.version).toEqual(1);
        expect(doc.languageId).toEqual(languageId);
        expect(doc.getText()).toEqual(initialText);
      });

      describe("textDocument/didChange", () => {
        it("should update the document", () => {
          onDidChange({
            textDocument: {
              uri,
              version: 2,
            },
            contentChanges: [
              {
                text: 'key: "new value"',
              },
            ],
          }, ctx);
          const doc = documents.get(uri);
          expect(doc.version).toEqual(2);
          expect(doc.getText()).toEqual('key: "new value"');
        });
      });

      describe("textDocument/didClose", () => {
        beforeEach(() => {
          onDidClose({ textDocument: { uri } }, ctx);
        });

        it("should remove the document from the map", () => {
          expect(() => documents.get(uri)).toThrow(
            `ENOENT: no Document found for ${uri}`,
          );
        });
      });
    });
  });
});
