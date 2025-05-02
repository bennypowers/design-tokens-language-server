import { beforeAll, beforeEach, describe, it } from 'jsr:@std/testing/bdd';
import { expect } from 'jsr:@std/expect';

import { FullTextDocument } from './textDocument.ts';

describe('FullTextDocument', () => {
  describe('given plain text', () => {
    const initialText = 'Hello, world!\nNew line.';
    const version = 0;
    const uri = 'file://test.txt';
    const languageId = 'txt';
    let doc = new FullTextDocument(uri, languageId, version, initialText);

    function resetDoc() {
      doc = new FullTextDocument(uri, languageId, version, initialText);
    }

    it('should create a FullTextDocument instance', () => {
      expect(doc).toBeInstanceOf(FullTextDocument);
    });

    describe('getText())', () => {
      it('should return the text of the document', () => {
        expect(doc.getText()).toBe(initialText);
      });
    });

    describe('getText(range)', () => {
      describe('with a range that covers the whole document', () => {
        it('should return the text of the document', () => {
          expect(
            doc.getText({
              start: { line: 0, character: 0 },
              end: { line: 1, character: 9 },
            }),
          ).toBe('Hello, world!\nNew line.');
        });
      });
      describe('with a range that covers a single line', () => {
        it('should return the text of the document', () => {
          expect(
            doc.getText({
              start: { line: 0, character: 0 },
              end: { line: 0, character: 5 },
            }),
          ).toBe('Hello');
        });
      });
    });

    describe('lineCount', () => {
      it('should return the length of the document', () => {
        expect(doc.lineCount).toBe(2);
      });
    });

    describe('languageId', () => {
      it('should return the languageId of the document', () => {
        expect(doc.languageId).toBe(languageId);
      });
    });

    describe('uri', () => {
      it('should return the languageId of the document', () => {
        expect(doc.uri).toBe(uri);
      });
    });

    describe('version', () => {
      it('should return the version of the document', () => {
        expect(doc.version).toBe(version);
      });
    });

    describe('positionAt()', () => {
      it('should return the position at the given offset', () => {
        expect(doc.positionAt(0)).toEqual({ line: 0, character: 0 });
        expect(doc.positionAt(5)).toEqual({ line: 0, character: 5 });
        expect(doc.positionAt(7)).toEqual({ line: 0, character: 7 });
        expect(doc.positionAt(13)).toEqual({ line: 0, character: 13 });
      });
    });

    describe('offsetAt()', () => {
      it('should return the offset at the given position', () => {
        expect(doc.offsetAt({ line: 0, character: 0 })).toBe(0);
        expect(doc.offsetAt({ line: 0, character: 5 })).toBe(5);
        expect(doc.offsetAt({ line: 0, character: 7 })).toBe(7);
        expect(doc.offsetAt({ line: 1, character: 2 })).toBe(16);
        expect(doc.offsetAt({ line: -1, character: 0 })).toBe(0);
        expect(doc.offsetAt({ line: Number.MAX_SAFE_INTEGER, character: 0 }))
          .toBe(initialText.length);
      });
    });

    describe('update()', () => {
      beforeEach(resetDoc);
      describe('with a full text change', () => {
        const newText = 'Hello, world!\nUpdated.';
        beforeEach(() => {
          doc.update([{ text: newText }], doc.version + 1);
        });
        it('should update the text of the document', () => {
          expect(doc.getText()).toBe(newText);
        });
        it('should return the updated length of the document', () => {
          expect(doc.lineCount).toBe(2);
        });
      });
      describe('with an incremental change', () => {
        const newText = 'Hello, world!\nUpdated.';
        beforeEach(() => {
          doc.update([{
            range: {
              start: { line: 1, character: 0 },
              end: { line: 1, character: 9 },
            },
            text: 'Updated.',
          }], doc.version + 1);
        });
        it('should update the text of the document', () => {
          expect(doc.getText()).toBe(newText);
        });
        it('should return the updated length of the document', () => {
          expect(doc.lineCount).toBe(2);
        });
        describe('then updating with another incremental change', () => {
          beforeEach(() => {
            doc.update([{
              range: {
                start: { line: 0, character: 0 },
                end: { line: 0, character: 5 },
              },
              text: 'Hi',
            }], doc.version + 1);
          });
          it('should update the text of the document', () => {
            expect(doc.getText()).toBe('Hi, world!\nUpdated.');
          });
        });
      });

      describe('with an empty change', () => {
        beforeEach(() => {
          doc.update([], doc.version + 1);
        });
        it('should not update the text of the document', () => {
          expect(doc.getText()).toBe('Hello, world!\nNew line.');
        });
        it('should not change the version of the document', () => {
          expect(doc.version).toBe(1);
        });
      });

      describe('with a change that adds lines', () => {
        const newText = 'Hello, world!\nUpdated.\nNew line.';
        beforeEach(() => {
          doc.update([{
            range: {
              start: { line: 1, character: 0 },
              end: { line: 1, character: 9 },
            },
            text: 'Updated.\nNew line.',
          }], doc.version + 1);
        });
        it('should update the text of the document', () => {
          expect(doc.getText()).toBe(newText);
        });
        it('should return the updated length of the document', () => {
          expect(doc.lineCount).toBe(3);
        });
      });

      describe('with multiple changes to a single line', () => {
        const newText = 'Hiya, world!\nNew line.';
        beforeEach(() => {
          doc.update([{
            range: {
              start: { line: 0, character: 0 },
              end: { line: 0, character: 4 },
            },
            text: 'Hi',
          }, {
            range: {
              start: { line: 0, character: 2 },
              end: { line: 0, character: 3 },
            },
            text: 'ya',
          }], doc.version + 1);
        });
        it('should update the text of the document', () => {
          expect(doc.getText()).toBe(newText);
        });
      });

      // handle the endLine - startLine === addedLineOffsets.length branch
      describe('with multiple, multiline edits', () => {
        const newText = 'Hello, world!\nUpdated.\nNew line.\nNew line.';
        beforeEach(() => {
          doc.update([{
            range: {
              start: { line: 0, character: 0 },
              end: { line: 1, character: 0 },
            },
            text: 'Hello, world!\nUpdated.\n',
          }, {
            range: {
              start: { line: 1, character: 0 },
              end: { line: 1, character: 8 },
            },
            text: 'Updated.\nNew line.',
          }], doc.version + 1);
        });
        it('should update the text of the document', () => {
          expect(doc.getText()).toBe(newText);
        });
      });
    });

    describe('FullTextDocument.applyEdits()', () => {
      beforeAll(resetDoc);
      describe('applying a single edit', () => {
        it('should apply single edit to the document', () => {
          expect(FullTextDocument.applyEdits(doc, [
            {
              range: {
                start: { line: 0, character: 0 },
                end: { line: 0, character: 5 },
              },
              newText: 'Hola',
            },
          ])).toBe('Hola, world!\nNew line.');
        });
      });

      describe('applying multiple edits', () => {
        it('should apply multiple edits to the document', () => {
          expect(FullTextDocument.applyEdits(doc, [
            {
              range: {
                start: { line: 0, character: 0 },
                end: { line: 0, character: 5 },
              },
              newText: 'Hi',
            },
            {
              range: {
                start: { line: 1, character: 0 },
                end: { line: 1, character: 9 },
              },
              newText: 'Updated.',
            },
          ])).toBe('Hi, world!\nUpdated.');
        });
      });

      describe('applying an edit which does not change the text', () => {
        it("should handle edits which don't change the text", () => {
          expect(FullTextDocument.applyEdits(doc, [
            {
              range: {
                start: { line: 0, character: 0 },
                end: { line: 0, character: 5 },
              },
              newText: 'Hello',
            },
          ])).toBe(initialText);
        });
      });

      describe('applying a single edit which removes text within a single line', () => {
        it('should apply the edit to the document', () => {
          expect(FullTextDocument.applyEdits(doc, [
            {
              range: {
                start: { line: 0, character: 0 },
                end: { line: 0, character: 5 },
              },
              newText: '',
            },
          ])).toBe(', world!\nNew line.');
        });
      });

      describe('applying multiple edits which remove text within a single line', () => {
        it('should apply the edit to the document', () => {
          expect(FullTextDocument.applyEdits(doc, [
            {
              range: {
                start: { line: 0, character: 0 },
                end: { line: 0, character: 5 },
              },
              newText: '',
            },
            {
              range: {
                start: { line: 0, character: 12 },
                end: { line: 0, character: 13 },
              },
              newText: '',
            },
          ])).toBe(', world\nNew line.');
        });
      });

      describe('attempting to apply multiple edits in which one edit is "before" the other', () => {
        it('should throw an error', () => {
          expect(() => {
            FullTextDocument.applyEdits(doc, [
              {
                range: {
                  start: { line: 0, character: 0 },
                  end: { line: 0, character: 2 },
                },
                newText: '',
              },
              {
                range: {
                  start: { line: 0, character: 1 },
                  end: { line: 0, character: 3 },
                },
                newText: '',
              },
            ]);
          }).toThrow('Overlapping edit');
        });
      });

      describe('applying an edit which removes text across multiple lines', () => {
        it('should apply the edit to the document', () => {
          expect(FullTextDocument.applyEdits(doc, [
            {
              range: {
                start: { line: 0, character: 0 },
                end: { line: 1, character: 0 },
              },
              newText: '',
            },
          ])).toBe('New line.');
        });
      });
    });
  });
});
