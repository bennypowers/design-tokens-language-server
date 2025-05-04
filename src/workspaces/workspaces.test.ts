import type * as LSP from 'vscode-languageserver-protocol';
import { afterAll, beforeAll, beforeEach, describe, it } from '@std/testing/bdd';

import { createTestContext, DTLSTestContext } from '#test-helpers';

import * as YAML from 'yaml';
import { Workspaces } from '#workspaces';
import { expect } from '@std/expect/expect';
import { toFileUrl } from '@std/path';

/** a comprehensive test suite for the Documents class */
describe('Workspaces', () => {
  let ctx: DTLSTestContext;
  let workspaces: Workspaces;
  let tmpDir: URL;

  beforeAll(async () => {
    tmpDir = toFileUrl(await Deno.makeTempDir() + '/');
    await Deno.writeTextFile(
      new URL('./package.json', tmpDir),
      JSON.stringify({
        name: 'Workspaces#test',
        version: '0.0.0',
        designTokensLanguageServer: {
          tokensFiles: [
            { path: 'tokens/*.json', prefix: 'token' },
            'tokens/*.yaml',
          ],
        },
      }),
    );
    await Deno.mkdir(new URL('./tokens/', tmpDir));
    await Deno.writeTextFile(
      new URL('./tokens/hooli.json', tmpDir),
      JSON.stringify({
        color: {
          $type: 'color',
          a: { $value: '#aaa' },
        },
      }),
    );
    await Deno.writeTextFile(
      new URL('./tokens/referer.yaml', tmpDir),
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
            { settings: { dtls: { prefix: 'hooli' } } },
            ctx,
          );
        });

        it('has no result', () => {
          expect(result).toBeUndefined();
        });

        it('applies settings to tokens', () => {
          expect(
            workspaces.getPrefixForUri(
              new URL('tokens/referer.yaml', tmpDir).href,
            ),
          )
            .toEqual('token');
        });

        it('adds workspaces', () => {
          expect(
            workspaces.getPrefixForUri(
              new URL('./tokens/referer.yaml', tmpDir).href,
            ),
          ).toEqual(
            'hooli',
          );
        });

        // describe("workspaces/didChangeWorkspaceFolders", () => {
        //   describe("called with the test package", () => {
        //     let result: void;
        //     beforeEach(async () => {
        //       const method =
        //         workspaces.handlers["workspace/didChangeWorkspaceFolders"];
        //       result = await method({
        //         event: {
        //           added: [{ name: "root", uri: tmpDir.href }],
        //           removed: [],
        //         },
        //       }, ctx);
        //     });
        //     it("has no result", () => {
        //       expect(result).toBeUndefined();
        //     });
        //     it("reloads files", () => {
        //       const refereryaml = new URL("./tokens/referer.yaml", tmpDir).href;
        //       expect(workspaces.getPrefixForUri(refereryaml))
        //         .toEqual("token");
        //     });
        //   });
        // });
      });
    });
  });
});
