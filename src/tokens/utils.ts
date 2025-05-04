import { DTLSClientSettings, TokenFile, TokenFileSpec } from '#lsp';
import { URI } from 'vscode-languageserver-protocol';

import { createRequire } from 'node:module';

function normalizePath(path: string, workspaceRoot: URI) {
  if (path.startsWith('~'))
    return path.replace('~', Deno.env.get('HOME')!);
  else if (path.startsWith('.'))
    return new URL(path, workspaceRoot).pathname;
  else if (path.startsWith('npm:')) {
    return createRequire(import.meta.url).resolve(path.replace('npm:', ''), {
      paths: [workspaceRoot],
    });
  } else {
    return path;
  }
}

export function normalizeTokenFile(
  tokenFile: TokenFile,
  workspaceRoot: URI,
  settings: Pick<DTLSClientSettings, 'prefix' | 'groupMarkers'> | null,
): TokenFileSpec {
  const tokenFilePath = typeof tokenFile === 'string' ? tokenFile : tokenFile.path;
  const tokenFilePrefix = typeof tokenFile === 'string' ? undefined : tokenFile.prefix;
  const tokenFileGroupMarkers = typeof tokenFile === 'string' ? undefined : tokenFile.groupMarkers;
  const path = normalizePath(tokenFilePath, workspaceRoot);
  const prefix = tokenFilePrefix || settings?.prefix;
  const groupMarkers = tokenFileGroupMarkers || settings?.groupMarkers;
  return {
    path,
    prefix,
    groupMarkers,
  };
}
