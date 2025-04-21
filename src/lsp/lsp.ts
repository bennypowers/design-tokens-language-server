import * as LSP from "vscode-languageserver-protocol";

import type { Server } from "#server";

import { Documents, documents } from "#css";
import { Logger } from "#logger";

import { TokenMap, tokens } from "#tokens";

import { initialize } from "./methods/initialize.ts";
import { documentColor } from "./methods/textDocument/documentColor.ts";
import { codeAction } from "./methods/textDocument/codeAction.ts";
import { diagnostic } from "./methods/textDocument/diagnostic.ts";
import { hover } from "./methods/textDocument/hover.ts";
import { completion } from "./methods/textDocument/completion.ts";
import { colorPresentation } from "./methods/textDocument/colorPresentation.ts";
import { resolve as completionItemResolve } from "./methods/completionItem/resolve.ts";
import { resolve as codeActionResolve } from "./methods/codeAction/resolve.ts";
import { didChangeConfiguration } from "./methods/workspace/didChangeConfiguration.ts";

const handlers = {
  ...documents.handlers,
  "initialize": initialize,
  "codeAction/resolve": codeActionResolve,
  "completionItem/resolve": completionItemResolve,
  "textDocument/codeAction": codeAction,
  "textDocument/colorPresentation": colorPresentation,
  "textDocument/completion": completion,
  "textDocument/diagnostic": diagnostic,
  "textDocument/documentColor": documentColor,
  "textDocument/hover": hover,
  "workspace/didChangeConfiguration": didChangeConfiguration,
};

export type RequestId = LSP.RequestMessage["id"];

export type SupportedMethod =
  | "initialized"
  | "$/cancelRequest"
  | "$/setTrace"
  | keyof typeof handlers;

export type SupportedParams = Parameters<
  typeof handlers[keyof typeof handlers]
>[0];

export type RequestTypeForMethod<M extends SupportedMethod> = M extends
  "initialized" ? { method: M; params: LSP.InitializedParams }
  : M extends "$/cancelRequest"
    ? { method: M; params: Pick<LSP.RequestMessage, "id"> }
  : M extends "$/setTrace" ? { method: M; params: LSP.SetTraceParams }
  : M extends keyof typeof handlers ? LSP.RequestMessage & {
      method: M;
      params: Parameters<typeof handlers[M]>[0];
    }
  : never;

export type ResultFor<M extends SupportedMethod> = Awaited<
  M extends "initialized" ? null
    : M extends "$/cancelRequest" ? null
    : M extends "$/setTrace" ? null
    : M extends keyof typeof handlers ? ReturnType<typeof handlers[M]>
    : never
>;

export type ResponseFor<M extends SupportedMethod> =
  & Omit<
    LSP.ResponseMessage,
    "result"
  >
  & { result: ResultFor<M> };

export interface TokenFileSpec {
  /**
   * The path to the token file, or a deno-compatible module specifier.
   * If the path is a relative path, it will be resolved relative to the
   * workspace root.
   * If it is a module specifier, it will be resolved relative to the
   * package.json file in the workspace root, i.e. in the node_modules folder.
   * @example ~/path/to/tokens.json
   * @example ./path/to/tokens.json
   * @example npm:package-name/path/to/tokens.json
   */
  path: string;
  /**
   * CSS variable name prefix to use for the token file.
   * if set, tokens in the file will be prefixed with this value.
   * @example "prefix": "my-design-system" => `--my-design-system-color-primary`
   */
  prefix?: string;
}

export type TokenFile = string | TokenFileSpec;

export interface DTLSClientSettings {
  dtls: {
    tokensFiles: TokenFile[];
  };
}

export interface DTLSContext {
  /**
   * Documents manager that represents the state of all documents.
   */
  documents: Documents;
  /**
   * All tokens available to the server.
   */
  tokens: TokenMap;
}

export interface DTLSContextWithLsp extends DTLSContext {
  /**
   * The LSP server protocol implementation.
   */
  lsp: Lsp;
}

type RequestMethodTypeGuard<M extends SupportedMethod> = (
  request: Omit<LSP.NotificationMessage, "jsonrpc">,
) => request is RequestTypeForMethod<M>;

function requestMethodTypeGuard<M extends SupportedMethod>(
  method: M,
): RequestMethodTypeGuard<M> {
  return ((request) => request.method === method) as RequestMethodTypeGuard<M>;
}

const isCancelRequest = requestMethodTypeGuard("$/cancelRequest");
const isSetTraceRequest = requestMethodTypeGuard("$/setTrace");
const isInitializedRequest = requestMethodTypeGuard("initialized");

/**
 * The Lsp class is responsible for processing LSP requests and notifications.
 * It handles the initialization of the server, and the processing of various LSP methods.
 */
export class Lsp {
  #handlerMap = new Map(Object.entries(handlers));
  #server: typeof Server;
  #resolveInitialized!: () => void;
  #initialized = new Promise<void>((r) => this.#resolveInitialized = r);
  #workspaceFolders = new Set<LSP.WorkspaceFolder>();
  #tokenFiles = new Set<TokenFile>();
  #initializationOptions?: LSP.LSPAny;
  #clientCapabilities?: LSP.ClientCapabilities;
  #traceLevel: LSP.TraceValues = LSP.TraceValues.Off;
  #cancelled = new Set<RequestId>();

  constructor(server: typeof Server) {
    this.#server = server;
  }

  async #updateConfiguration(context: DTLSContext) {
    for (const { uri } of this.#workspaceFolders) {
      const pkgJsonPath = new URL("./package.json", `${uri}/`);
      const mod = await import(pkgJsonPath.href, { with: { type: "json" } });
      for (
        const tokensFile
          of mod.default?.designTokensLanguageServer?.tokensFiles ??
            []
      ) {
        this.#tokenFiles.add(tokensFile);
      }
    }

    for (const tokensFile of this.#tokenFiles) {
      await context.tokens.register(tokensFile);
    }
  }

  #cancelRequest(request: LSP.RequestMessage) {
    const { id } = request;
    this.#cancelled.add(id);
    Logger.debug`📵 Cancel ${id}`;
    return null;
  }

  #setTrace(value: LSP.TraceValues) {
    Logger.info`Set trace level to ${value}`;
    this.#traceLevel = value;
    return null;
  }

  /**
   * Caches the workspace folders, initialization options, and client capabilities.
   *
   * @param params - The parameters for the initialization request.
   * @param context - The context for the server.
   */
  public async initialize(params: LSP.InitializeParams, context: DTLSContext) {
    const { capabilities, workspaceFolders, initializationOptions, trace } =
      params;
    if (trace) this.#setTrace(trace);
    for (const dir of workspaceFolders ?? []) this.#workspaceFolders.add(dir);
    this.#clientCapabilities = capabilities;
    this.#initializationOptions = initializationOptions;
    await this.#updateConfiguration(context);
  }

  /**
   * Synchronize the server with client settings
   */
  public async updateSettings(
    settings: DTLSClientSettings,
    context: DTLSContext,
  ) {
    for (const tokenFile of settings?.dtls?.tokensFiles ?? []) {
      this.#tokenFiles.add(tokenFile);
    }
    await this.#updateConfiguration(context);
  }

  /**
   * A promise which resolves when the server has completed initialization.
   */
  public initialized() {
    return this.#initialized;
  }

  /**
   * Informs the caller whether or not the request has been cancelled.
   */
  public isCancelledRequest(id: RequestId) {
    return this.#cancelled.has(id);
  }

  /**
   * Requests client configuration for a specific section.
   */
  public requestConfiguration(scopeUri: LSP.URI, section: string) {
    return this.#server.push({
      method: "workspace/configuration",
      params: {
        items: [
          {
            scopeUri,
            section,
          },
        ],
      },
    });
  }

  /**
   * Processes the given request and returns the result.
   *
   * @param request - The request to process.
   * @returns The result of the request.
   */
  public async process(request: LSP.RequestMessage): Promise<unknown> {
    const context = { lsp: this, documents, tokens };
    if (LSP.Message.isRequest(request) && this.#cancelled.has(request.id)) {
      return null;
    } else if (isInitializedRequest(request)) {
      await this.#updateConfiguration(context);
      return this.#resolveInitialized();
    } else if (isSetTraceRequest(request)) {
      return this.#setTrace(request.params.value);
    } else if (isCancelRequest(request)) {
      return this.#cancelRequest(request);
    } else if (request.method) {
      const method = this.#handlerMap.get(request.method);
      // deno-lint-ignore no-explicit-any
      return await method?.(request.params as any, context) ?? null;
    }
  }
}
