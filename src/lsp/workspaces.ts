import * as LSP from "vscode-languageserver-protocol";

import { Logger } from "#logger";

import {
  DTLSClientSettings,
  DTLSContext,
  DTLSContextWithWorkspaces,
  TokenFile,
  TokenFileSpec,
} from "#lsp";

import { JsonDocument } from "#json";

import { normalizeTokenFile } from "../tokens/utils.ts";
import { isGlob } from "@std/path";
import { expandGlob } from "@std/fs/expand-glob";

const decoder = new TextDecoder();

function logSpecAdd(spec: TokenFileSpec) {
  const { prefix, path, groupMarkers } = spec;
  Logger.debug`🪙 Adding token spec`;
  Logger.debug`  from ${path}`;
  if (prefix) Logger.debug`  with prefix ${prefix}`;
  if (groupMarkers) Logger.debug`  and groupMarkers ${groupMarkers}`;
}

async function tryToLoadSettingsFromPackageJson(
  uri: LSP.DocumentUri,
): Promise<DTLSClientSettings | null> {
  try {
    const pkgJsonPath = new URL("./package.json", `${uri}/`);
    Logger.debug`🎒 Loading package.json from ${pkgJsonPath.href}`;
    const mod = await import(pkgJsonPath.href, { with: { type: "json" } });
    Logger
      .debug`  ...loaded package.json for ${mod.default.name}@${mod.default.version}`;
    const settings = mod.default?.designTokensLanguageServer;
    return settings;
  } catch (e) {
    if (e instanceof SyntaxError) {
      Logger.error`Could not load package.json: ${e}`;
    }
    throw e;
  }
}

/**
 * Manages LSP workspace folders and settings.
 * - reads workspace configuration from the workspace folder's package.json
 *   and registers the tokens files by resolving them against the workspace dir
 * - watches for changes in the workspace configuration
 * - handles workspace/didChangeConfiguration
 */
export class Workspaces {
  #settings: { dtls: DTLSClientSettings } | null = null;
  #tokenSpecs = new Set<TokenFileSpec>();
  #workspaces = new Set<LSP.WorkspaceFolder>();

  /**
   * Normalizes the settings by merging them with the default settings.
   */
  #normalizeSettings(settings: DTLSClientSettings | null) {
    const clone = structuredClone<Partial<DTLSClientSettings>>(settings ?? {});
    clone.prefix ||= this.#settings?.dtls?.prefix;
    clone.groupMarkers ||= this.#settings?.dtls?.groupMarkers;
    return clone;
  }

  /**
   * Normalizes the token file path and prefix.
   * inherits the prefix from the settings if not provided in the spec
   * and from the workspace folder if not provided by the project
   */
  #normalizeTokenFile(
    tokenFile: TokenFile,
    workspaceRoot: LSP.URI,
    settings: DTLSClientSettings | null,
  ): TokenFileSpec {
    return normalizeTokenFile(
      tokenFile,
      workspaceRoot,
      this.#normalizeSettings(settings),
    );
  }

  async #loadSpec(context: DTLSContext, spec: TokenFileSpec) {
    logSpecAdd(spec);
    this.#tokenSpecs.add(spec);
    try {
      const tokenfileContent = decoder.decode(await Deno.readFile(spec.path));
      const uri = `file://${spec.path.replace("file://", "")}`;
      const doc = JsonDocument.create(context, uri, tokenfileContent);
      context.documents.add(doc);
    } catch (e) {
      Logger.error`Could not read token file ${spec.path}: ${
        (e as Error).message
      }`;
      this.#tokenSpecs.delete(spec);
    }
  }

  async #updateWorkspaceSettings(
    context: DTLSContext,
    uri: string,
    settings: DTLSClientSettings | null,
  ) {
    for (const file of settings?.tokensFiles ?? []) {
      const spec = this.#normalizeTokenFile(file, uri, settings);
      if (isGlob(spec.path)) {
        const specs = expandGlob(spec.path, { includeDirs: false });
        for await (const { path } of specs) {
          await this.#loadSpec(context, { ...spec, path });
        }
      } else {
        await this.#loadSpec(context, spec);
      }
    }
  }

  async #updateConfiguration(context: DTLSContext) {
    for (const ws of this.#workspaces) {
      const settings = await tryToLoadSettingsFromPackageJson(ws.uri);
      await this.#updateWorkspaceSettings(context, ws.uri, settings);
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
    const { settings } = params;
    Logger.debug`User settings ${settings}`;
    this.#settings = settings;
    const uri = settings.workspaceRoot ?? "";
    await this.#updateWorkspaceSettings(context, uri, settings);
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
    for (const folder of folders ?? []) {
      Logger.debug`📁 Adding workspace folder ${folder.name}@${folder.uri}`;
      this.#workspaces.add(folder);
    }
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
