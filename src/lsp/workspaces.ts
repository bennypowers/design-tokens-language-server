import * as LSP from "vscode-languageserver-protocol";

import { Logger } from "#logger";

import {
  DTLSClientSettings,
  DTLSContext,
  DTLSContextWithWorkspaces,
  TokenFile,
  TokenFileSpec,
} from "#lsp";
import { createRequire } from "node:module";

/**
 * Manages LSP workspace folders and settings.
 * - reads workspace configuration from the workspace folder's package.json
 *   and registers the tokens files by resolving them against the workspace dir
 * - watches for changes in the workspace configuration
 * - handles workspace/didChangeConfiguration
 */
export class Workspaces {
  #settings: DTLSClientSettings | null = null;
  #tokenSpecs = new Set<TokenFileSpec>();
  #workspaces = new Set<LSP.WorkspaceFolder>();

  #normalizePath(path: string, workspaceRoot: LSP.URI) {
    if (path.startsWith("~")) {
      return path.replace("~", Deno.env.get("HOME")!);
    } else if (path.startsWith(".")) {
      return path.replace(".", Deno.cwd());
    } else if (path.startsWith("npm:")) {
      const require = createRequire(import.meta.url);
      return require.resolve(path.replace("npm:", ""), {
        paths: [workspaceRoot],
      });
    } else {
      return path;
    }
  }

  /**
   * Normalizes the token file path and prefix.
   * inherits the prefix from the settings if not provided in the spec
   * and from the workspace folder if not provided by the project
   */
  #normalizeTokenFile(
    tokenFile: TokenFile,
    workspaceRoot: LSP.URI,
    settings: DTLSClientSettings["dtls"] | null,
  ): TokenFileSpec {
    const tokenFilePath = typeof tokenFile === "string"
      ? tokenFile
      : tokenFile.path;
    const tokenFilePrefix = typeof tokenFile === "string"
      ? undefined
      : tokenFile.prefix;
    const tokenFileGroupMarkers = typeof tokenFile === "string"
      ? undefined
      : tokenFile.groupMarkers;
    const path = this.#normalizePath(tokenFilePath, workspaceRoot);
    const prefix = tokenFilePrefix ||
      settings?.prefix ||
      this.#settings?.dtls?.prefix;
    const groupMarkers = tokenFileGroupMarkers ||
      settings?.groupMarkers ||
      this.#settings?.dtls?.groupMarkers;
    return {
      path,
      prefix,
      groupMarkers,
    };
  }

  async #updateConfiguration(context: DTLSContext) {
    for (const ws of this.#workspaces) {
      const { uri } = ws;
      const pkgJsonPath = new URL("./package.json", `${uri}/`);
      const mod = await import(pkgJsonPath.href, { with: { type: "json" } });
      const settings = mod.default?.designTokensLanguageServer;
      for (const file of settings?.tokensFiles ?? []) {
        const spec = this.#normalizeTokenFile(file, uri, settings);
        Logger
          .debug`Token file ${spec.path} with prefix ${spec.prefix} and group markers ${spec.groupMarkers}`;
        this.#tokenSpecs.add(spec);
      }
    }

    for (const tokensFile of this.#tokenSpecs) {
      await context.tokens.register(tokensFile);
    }
  }

  /**
   * Handle the `workspace/didChangeConfiguration` request.
   *
   * This request is sent by the client to notify the server about changes in
   * configuration settings.
   *
   * @param params - The parameters of the request, including the changed
   * configuration settings.
   * @param context - The context of the server, including the workspace and
   * documents.
   * @returns A promise that resolves when the configuration change is handled.
   */
  #didChangeConfiguration = async (
    params: LSP.DidChangeConfigurationParams,
    context: DTLSContextWithWorkspaces,
  ) => {
    Logger.debug`User settings ${params.settings}`;
    this.#settings = params.settings;
    for (const file of params.settings?.dtls?.tokensFiles ?? []) {
      const spec = this.#normalizeTokenFile(
        file,
        params.settings?.workspaceRoot ?? "",
        params.settings,
      );
      this.#tokenSpecs.add(spec);
    }
    await this.#updateConfiguration(context);
  };

  /**
   * Adds the given workspace folder to the list of workspaces.
   * watches for changes to the workspace folder's package.json
   * and updates the tokens map accordingly
   */
  public async add(
    folders: LSP.WorkspaceFolder[] | null | undefined,
    context: DTLSContext,
  ) {
    for (const folder of folders ?? []) this.#workspaces.add(folder);
    await this.#updateConfiguration(context);
  }

  public async initialize(context: DTLSContext) {
    await this.#updateConfiguration(context);
  }

  public get handlers() {
    return {
      "workspace/didChangeConfiguration": this.#didChangeConfiguration,
    };
  }
}
