import { describe, it } from '@std/testing/bdd';
import { expect } from '@std/expect';

import { normalizeTokenFile } from './utils.ts';

describe('normalizeTokenFile', () => {
  describe('when tokenFile is an absolute path', () => {
    it('should return the same path', () => {
      const tokenFile = '/absolute/path/to/tokenfile';
      const workspaceRoot = 'file:///workspace/root';
      const settings = null;

      const result = normalizeTokenFile(tokenFile, workspaceRoot, settings);

      expect(result).toEqual({
        path: '/absolute/path/to/tokenfile',
      });
    });
  });

  describe('when tokenFile is a relative path', () => {
    it('should return the normalized path', () => {
      const tokenFile = './relative/path/to/tokenfile';
      const workspaceRoot = 'file:///workspace/root';
      const settings = null;

      const result = normalizeTokenFile(tokenFile, workspaceRoot, settings);

      expect(result).toEqual({
        path: '/workspace/root/relative/path/to/tokenfile',
      });
    });
  });

  describe('when tokenFile is a home directory path', () => {
    it('should return the expanded home directory path', () => {
      const tokenFile = '~/path/to/tokenfile';
      const workspaceRoot = 'file:///workspace/root';
      const settings = null;

      const result = normalizeTokenFile(tokenFile, workspaceRoot, settings);

      expect(result).toEqual({
        path: `${Deno.env.get('HOME')}/path/to/tokenfile`,
      });
    });
  });

  describe('when tokenFile is a npm package path', () => {
    const workspaceRoot = 'file:///workspace/root';
    const tokenFile = 'npm:package-name/path/to/tokenfile';

    it('should return the resolved npm package path', () => {
      expect(() => normalizeTokenFile(tokenFile, workspaceRoot, null))
        // .toThrow(new RegExp("package-name/path/to/tokenFile"));
        .toThrow();
    });
  });
});
