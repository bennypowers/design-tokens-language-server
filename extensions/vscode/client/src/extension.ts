import * as path from "node:path";
import * as os from "node:os";
import { ExtensionContext } from "vscode";

import {
  LanguageClient,
  LanguageClientOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient;

export async function activate(context: ExtensionContext) {
  const platform = os.platform();
  const arch = os.arch();

  const args = ["--stdio"];

  // Determine the OS-specific binary name
  // Uses standard platform naming: linux-x64, darwin-arm64, win32-x64, etc.
  const binaryName = (() => {
    const archMapping: Record<string, string> = {
      arm64: "arm64",
      x64: "x64",
    };

    const osMapping: Record<string, string> = {
      darwin: "darwin",
      linux: "linux",
      win32: "win32",
    };

    const archSuffix = archMapping[arch];
    const osSuffix = osMapping[platform];

    if (!archSuffix || !osSuffix) {
      throw new Error(
        `Unsupported platform or architecture: ${platform}-${arch}`,
      );
    }

    const ext = platform === "win32" ? ".exe" : "";
    return `design-tokens-language-server-${osSuffix}-${archSuffix}${ext}`;
  })();

  const command = context.asAbsolutePath(
    path.join("dist", "bin", binaryName),
  );

  const clientOptions: LanguageClientOptions = {
    documentSelector: [
      { scheme: "file", language: "css" },
      { scheme: "file", language: "json" },
      { scheme: "file", language: "yaml" },
    ],
  };

  client = new LanguageClient(
    "design-tokens-language-server",
    "Design Tokens Language Server",
    {
      run: { command, args, transport: TransportKind.stdio },
      debug: { command, args, transport: TransportKind.stdio },
    },
    clientOptions,
  );

  try {
    await client.start();
  } catch (error) {
    console.error(error);
  }
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}
