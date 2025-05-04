import * as LSP from 'vscode-languageserver-protocol';
import { beforeEach, describe, it } from '@std/testing/bdd';
import { expect } from '@std/expect';

import { Documents } from '#documents';
import { Workspaces } from '#workspaces';
import { Tokens } from '#tokens';

import { Lsp } from '#lsp';
import { Server } from '#server';

describe('Lsp', () => {
  describe('with default options', () => {
    let lsp: Lsp;

    beforeEach(() => {
      const documents = new Documents();
      const workspaces = new Workspaces(Server);
      const tokens = new Tokens();
      lsp = new Lsp(documents, workspaces, tokens);
    });

    it('should create an instance of Lsp', () => {
      expect(lsp).toBeInstanceOf(Lsp);
    });

    describe("'initialize'", () => {
      let initializeResult: LSP.InitializeResult;
      beforeEach(async () => {
        initializeResult = await lsp.process({
          jsonrpc: '2.0',
          id: 0,
          method: 'initialize',
          params: {
            processId: 1000,
            rootUri: 'file:///path/to/root',
            capabilities: {},
          } satisfies LSP.InitializeParams,
        }) as LSP.InitializeResult;
      });
      it('should return an InitializeResult', () => {
        expect(initializeResult).toEqual(
          {
            serverInfo: {
              name: 'design-tokens-language-server',
              version: initializeResult.serverInfo?.version,
            },
            capabilities: {
              codeActionProvider: {
                codeActionKinds: [
                  LSP.CodeActionKind.QuickFix,
                  LSP.CodeActionKind.RefactorRewrite,
                  LSP.CodeActionKind.SourceFixAll,
                ],
                resolveProvider: true,
              },
              colorProvider: true,
              completionProvider: {
                completionItem: {
                  labelDetailsSupport: true,
                },
                resolveProvider: true,
              },
              diagnosticProvider: {
                interFileDependencies: false,
                workspaceDiagnostics: false,
              },
              textDocumentSync: LSP.TextDocumentSyncKind.Incremental,
              referencesProvider: true,
              hoverProvider: true,
              definitionProvider: true,
            },
          } satisfies LSP.InitializeResult,
        );
      });
    });
  });
});
