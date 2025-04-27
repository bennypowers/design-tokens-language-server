import * as path from "node:path";
import * as os from "node:os";
import { ExtensionContext, window } from "vscode";

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
  const binaryName = (() => {
    const archMapping: Record<string, string> = {
      arm64: "aarch64",
      x64: "x86_64",
    };

    const osMapping: Record<string, string> = {
      darwin: "apple-darwin",
      linux: "unknown-linux-gnu",
      win32: "pc-windows-msvc.exe",
    };

    const architecture = archMapping[arch];
    const operatingSystem = osMapping[platform];

    if (!architecture || !operatingSystem) {
      throw new Error(
        `Unsupported platform or architecture: ${platform}-${arch}`,
      );
    }

    return `design-tokens-language-server-${architecture}-${operatingSystem}`;
  })();

  const command = context.asAbsolutePath(
    path.join("dist", "bin", binaryName),
  );

  const clientOptions: LanguageClientOptions = {
    documentSelector: [
      { scheme: "file", language: "css" },
      { scheme: "file", language: "json" },
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
