import { beforeEach, describe, it } from '@std/testing/bdd';
import { expect } from '@std/expect';

import { CompletionList, Position, TextDocumentIdentifier } from 'vscode-languageserver-protocol';

import { createTestContext, DTLSTestContext } from '#test-helpers';

import { completion } from './completion.ts';
import { DTLSTextDocument } from '#document';
import { CssDocument } from '#css';

function getCompletionsForWord(
  ctx: DTLSTestContext,
  word: string,
  content: string,
  language: DTLSTextDocument['language'] = 'css',
) {
  const textDocument = ctx.documents.createDocument(language, content);
  const doc = ctx.documents.get(textDocument.uri);
  const position = doc.positionForSubstring(word, 'end');
  return completion({ textDocument, position }, ctx);
}

describe('textDocument/completion', () => {
  let ctx: DTLSTestContext;

  beforeEach(async () => {
    ctx = await createTestContext({
      testTokensSpecs: [
        {
          prefix: 'token',
          spec: 'file:///tokens.json',
          tokens: {
            color: {
              $type: 'color',
              red: {
                _: {
                  $value: '#ff0000',
                },
                hex: {
                  $value: '#ff0000',
                },
              },
            },
            space: {
              $type: 'size',
              small: {
                $value: '4px',
              },
            },
            font: {
              weight: {
                $type: 'fontWeight',
                thin: {
                  $value: 100,
                },
              },
            },
          },
        },
      ],
    });
  });

  describe('in an empty document', () => {
    let textDocument: TextDocumentIdentifier;
    beforeEach(() => {
      textDocument = ctx.documents.createDocument('css', '');
    });
    it('should return no completions', () => {
      const completions = completion({
        textDocument,
        position: { line: 0, character: 0 },
      }, ctx);
      expect(completions).toBeNull();
    });
  });

  describe('in a document with a css rule', () => {
    let textDocument: TextDocumentIdentifier;
    let doc: CssDocument;
    beforeEach(() => {
      textDocument = ctx.documents.createDocument(
        'css',
        /*css*/ `
          body {
            a`,
      );
      doc = ctx.documents.get(textDocument.uri) as CssDocument;
    });
    it('should return no completions', () => {
      const completions = completion({
        textDocument,
        position: doc.positionForSubstring('a', 'end'),
      }, ctx);

      expect(completions).toBeNull();
    });
  });

  describe('adding the token prefix in a malformed block', () => {
    let textDocument: TextDocumentIdentifier;
    let doc: CssDocument;
    beforeEach(() => {
      textDocument = ctx.documents.createDocument(
        'css',
        /*css*/ `
        body {
          token
        }
      `,
      );
      doc = ctx.documents.get(textDocument.uri) as CssDocument;
    });
    it('should return all token completions', () => {
      const completions = completion({
        textDocument,
        position: doc.positionForSubstring('token', 'end'),
      }, ctx);
      expect(completions?.items).toHaveLength(ctx.tokens.size);
    });
  });

  describe('adding the token prefix as a property name', () => {
    let textDocument: TextDocumentIdentifier;
    let doc: CssDocument;
    let completions: CompletionList | null;
    beforeEach(() => {
      textDocument = ctx.documents.createDocument(
        'css',
        /*css*/ `
      body {
        --token
      }
    `,
      );
      doc = ctx.documents.get(textDocument.uri) as CssDocument;
      completions = completion({
        textDocument,
        position: doc.positionForSubstring('--token', 'end'),
      }, ctx);
    });
    it('should return all token completions', () => {
      expect(completions?.items).toHaveLength(ctx.tokens.size);
    });
    it('should return token completions as property names', () => {
      for (const item of completions?.items ?? [])
        expect(item.textEdit?.newText).toMatch(/^--token/);
    });
  });

  describe('adding the token prefix as a property value', () => {
    let textDocument: TextDocumentIdentifier;
    let doc: CssDocument;
    let completions: CompletionList | null;
    beforeEach(() => {
      textDocument = ctx.documents.createDocument(
        'css',
        /*css*/ `
          body {
            color: token
          }
        `,
      );
      doc = ctx.documents.get(textDocument.uri) as CssDocument;
      completions = completion({
        textDocument,
        position: doc.positionForSubstring('token', 'end'),
      }, ctx);
    });
    it('should return all token completions', () => {
      expect(completions?.items).toHaveLength(ctx.tokens.size);
    });
    it('should return token completions as var() calls', () => {
      for (const item of completions?.items ?? [])
        expect(item.textEdit?.newText).toMatch(/^var\(--token/);
    });
  });
});
