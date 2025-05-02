import * as LSP from "vscode-languageserver-protocol";

import { Logger } from "#logger";

import {
  DTLSClientSettings,
  DTLSContext,
  TokenFile,
  TokenFileSpec,
} from "#lsp";

import { JsonDocument } from "#json";
import { YamlDocument } from "#yaml";

import { normalizeTokenFile } from "../tokens/utils.ts";
import { isGlob, toFileUrl } from "@std/path";
import { expandGlob } from "@std/fs/expand-glob";
import { DTLSDocument } from "#documents";

const decoder = new TextDecoder();

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
  public get handlers() {
    return {
      "workspace/didChangeConfiguration": this.#didChangeConfiguration,
      "workspace/didChangeWorkspaceFolders": this.#didChangeWorkspaceFolders,
    };
  }

  #specs = new Map<LSP.DocumentUri, TokenFileSpec>();
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

  #loadedSpecs = new Set();

  async #loadSpec(
    context: DTLSContext,
    spec: TokenFileSpec,
    { force = false } = {},
  ) {
    const { prefix, path, groupMarkers } = spec;
    if (!force && this.#loadedSpecs.has(path)) return;
    Logger.debug`🪙 Adding token spec`;
    Logger.debug`  from ${path}`;
    if (prefix) Logger.debug`  with prefix ${prefix}`;
    if (groupMarkers) Logger.debug`  and groupMarkers ${groupMarkers}`;
    this.#tokenSpecs.add(spec);
    try {
      const tokenfileContent = decoder.decode(await Deno.readFile(spec.path));
      const uri = toFileUrl(spec.path.replace("file://", "")).href;
      const language = uri.split(".").pop()?.replace("yml", "yaml");
      if (!language) throw new Error(`Could not identify language for ${uri}`);
      let doc: DTLSDocument;
      switch (language) {
        case "json":
          doc = JsonDocument.create(context, uri, tokenfileContent);
          break;
        case "yaml":
          doc = YamlDocument.create(context, uri, tokenfileContent);
          break;
        default:
          throw new Error(`Unknown language: ${language}`);
      }
      this.#specs.set(uri, spec);
      this.#loadedSpecs.add(spec.path);
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
      const normalizedButGlobby = this.#normalizeTokenFile(file, uri, settings);
      if (isGlob(normalizedButGlobby.path)) {
        const specs = expandGlob(normalizedButGlobby.path, {
          includeDirs: false,
        });
        for await (const { path } of specs) {
          await this.#loadSpec(context, { ...normalizedButGlobby, path });
        }
      } else {
        await this.#loadSpec(context, normalizedButGlobby);
      }
    }
  }

  async #updateConfiguration(
    context: DTLSContext,
    { force }: { force: boolean },
  ) {
    for (const ws of this.#workspaces) {
      Logger.debug`📁 Adding workspace folder ${ws.name}@${ws.uri}`;
      const settings = await tryToLoadSettingsFromPackageJson(ws.uri);
      await this.#updateWorkspaceSettings(context, ws.uri, settings);
    }

    for (const tokensFile of this.#tokenSpecs) {
      await context.tokens.register(tokensFile, { force });
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
    context: DTLSContext,
  ) => {
    const { settings } = params;
    Logger.debug`User settings ${settings}`;
    this.#settings = settings;
    const uri = settings.workspaceRoot ?? "";
    await this.#updateWorkspaceSettings(context, uri, settings);
    await this.#updateConfiguration(context, { force: true });
  };

  /**
   * Adds the given workspace folder to the list of workspaces.
   * watches for changes to the workspace folder's package.json
   * and updates the tokens map accordingly
   */
  #didChangeWorkspaceFolders = async (
    params: LSP.DidChangeWorkspaceFoldersParams,
    context: DTLSContext,
  ) => {
    for (const folder of params.event.removed) {
      this.#workspaces.delete(folder);
    }
    for (const folder of params.event.added) {
      this.#workspaces.add(folder);
    }
    await this.#updateConfiguration(context, { force: false });
  };

  /**
   * Get the configured token prefix for a given document uri
   */
  public getPrefixForUri(uri: LSP.DocumentUri) {
    return this.#specs.get(uri)?.prefix ?? null;
  }

  /**
   * Get the token file spec for a given document uri
   */
  public getSpecForUri(uri: LSP.DocumentUri) {
    return this.#specs.get(uri) ?? null;
  }

  protected _addSpec(uri: LSP.DocumentUri, spec: TokenFileSpec) {
    this.#specs.set(uri, spec);
  }

  public async initialize(context: DTLSContext) {
    await this.#updateConfiguration(context, { force: false });
  }
}
