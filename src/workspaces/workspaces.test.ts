import type * as LSP from 'vscode-languageserver-protocol';
import { afterAll, afterEach, beforeAll, beforeEach, describe, it } from '@std/testing/bdd';

import { createTestContext, DTLSTestContext } from '#test-helpers';

import * as YAML from 'yaml';
import { Workspaces } from '#workspaces';
import { expect } from '@std/expect/expect';
import { join, toFileUrl } from '@std/path';

/** a comprehensive test suite for the Documents class */
describe('Workspaces', () => {
  let ctx: DTLSTestContext;
  let workspaces: Workspaces;
  let tmpDir: URL;
  let packageUri: URL;
  let hooliUri: URL;
  let refererUri: URL;

  beforeAll(async () => {
    tmpDir = toFileUrl(await Deno.makeTempDir() + '/');

    packageUri = new URL('./package.json', tmpDir);
    hooliUri = new URL('./tokens/tokens.json', tmpDir);
    refererUri = new URL('./tokens/referer.yaml', tmpDir);

    const tokensUri = new URL('./tokens/', tmpDir);

    await Deno.mkdir(tokensUri, { recursive: true });

    await Deno.writeTextFile(
      packageUri,
      JSON.stringify({
        name: 'Workspaces#test',
        version: '0.0.0',
        designTokensLanguageServer: {
          tokensFiles: [
            { path: './tokens/*.json', prefix: 'token' },
            './tokens/*.yaml',
          ],
        },
      }),
    );

    await Deno.writeTextFile(
      hooliUri,
      JSON.stringify({
        color: {
          $type: 'color',
          a: { $value: '#aaa' },
        },
      }),
    );

    await Deno.writeTextFile(
      refererUri,
      YAML.stringify({
        color: {
          $type: 'color',
          b: { $value: '{color.a}' },
        },
      }),
    );
  });

  afterAll(async () => {
    await Deno.remove(tmpDir, { recursive: true });
  });

  beforeEach(async () => {
    ctx = await createTestContext({ workspaceRoot: tmpDir.href });
    workspaces = ctx.workspaces;
  });

  describe('add', () => {
    beforeEach(async () => {
      await workspaces.add(ctx, { uri: tmpDir.href, name: 'root' });
    });

    describe('workspaces/didChangeConfiguration', () => {
      describe('called with some fallback settings', () => {
        let result: void;
        beforeEach(async () => {
          const method = workspaces.handlers['workspace/didChangeConfiguration'];
          result = await method(
            { settings: { dtls: { prefix: 'global' } } },
            ctx,
          );
        });

        it('has no result', () => {
          expect(result).toBeUndefined();
        });

        it('applies settings to tokens', () => {
          expect(workspaces.getSpecForUri(refererUri.href))
            .toEqual({
              path: join(tmpDir.pathname, 'tokens', 'referer.yaml'),
            });
          expect(workspaces.getSpecForUri(hooliUri.href)).toEqual({
            path: join(tmpDir.pathname, 'tokens', 'tokens.json'),
            prefix: 'token',
          });
        });
      });
    });
  });
});
