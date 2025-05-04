import * as LSP from "vscode-languageserver-protocol";

import { Documents } from "#documents";
import { Logger } from "#logger";
import { Tokens } from "#tokens";
import { Workspaces } from "#workspaces";

import * as DocumentColor from "./methods/textDocument/documentColor.ts";
import * as CodeAction from "./methods/textDocument/codeAction.ts";
import * as Diagnostic from "./methods/textDocument/diagnostic.ts";
import * as Hover from "./methods/textDocument/hover.ts";
import * as Completion from "./methods/textDocument/completion.ts";
import { colorPresentation } from "./methods/textDocument/colorPresentation.ts";
import { resolve as completionItemResolve } from "./methods/completionItem/resolve.ts";
import { resolve as codeActionResolve } from "./methods/codeAction/resolve.ts";
import * as Definition from "./methods/textDocument/definition.ts";
import * as References from "./methods/textDocument/references.ts";

import manifest from "../../package.json" with { type: "json" };
import { Server } from "#server";

const { version } = manifest;

const handlers = {
  "codeAction/resolve": codeActionResolve,
  "completionItem/resolve": completionItemResolve,
  "textDocument/codeAction": CodeAction.codeAction,
  "textDocument/definition": Definition.definition,
  "textDocument/colorPresentation": colorPresentation,
  "textDocument/completion": Completion.completion,
  "textDocument/diagnostic": Diagnostic.diagnostic,
  "textDocument/documentColor": DocumentColor.documentColor,
  "textDocument/hover": Hover.hover,
  "textDocument/references": References.references,
};

type Handlers =
  & typeof handlers
  & { initialize: Lsp["initialize"] }
  & Documents["handlers"]
  & Workspaces["handlers"];

type Handler = Handlers[keyof Handlers];

type RequestMethodTypeGuard<M extends SupportedMethod> = (
  request: Omit<LSP.NotificationMessage, "jsonrpc">,
) => request is RequestTypeForMethod<M>;

export type RequestId = LSP.RequestMessage["id"];

export type SupportedMethod =
  | "initialized"
  | "$/cancelRequest"
  | "$/setTrace"
  | keyof Handlers;

export type SupportedParams = Parameters<Handler>[0];

export type RequestTypeForMethod<M extends SupportedMethod> = M extends
  "initialized" ? { method: M; params: LSP.InitializedParams }
  : M extends "$/cancelRequest"
    ? { method: M; params: Pick<LSP.RequestMessage, "id"> }
  : M extends "$/setTrace" ? { method: M; params: LSP.SetTraceParams }
  : M extends keyof Handlers ? LSP.RequestMessage & {
      method: M;
      params: Parameters<Handlers[M]>[0];
    }
  : never;

export type ResultFor<M extends SupportedMethod> = Awaited<
  M extends "initialized" ? null
    : M extends "$/cancelRequest" ? null
    : M extends "$/setTrace" ? null
    : M extends keyof Handlers ? ReturnType<Handlers[M]>
    : never
>;

export type ResponseFor<M extends SupportedMethod> =
  & Omit<
    LSP.ResponseMessage,
    "result"
  >
  & { result: ResultFor<M> };

type HandlerFor<M extends SupportedMethod> = (
  params: M extends keyof Handlers ? Parameters<Handlers[M]>[0]
    : never,
  context: DTLSContext,
) => Promise<ResultFor<M>> | ResultFor<M> | null;

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

  /**
   * Terminal token path path which signifies that a token is also a group.
   *
   * @see https://github.com/design-tokens/community-group/issues/97
   * @see https://github.com/amzn/style-dictionary/issues/716
   */
  groupMarkers?: string[];
}

export type TokenFile = string | TokenFileSpec;

export interface DTLSClientSettings {
  /**
   * List of token file paths or spec objects to load
   */
  tokensFiles: TokenFile[];

  /**
   * CSS variable name prefix to use for the token file.
   * if set, tokens in the file will be prefixed with this value.
   *
   * Applies to all tokens in the project.
   * @example "prefix": "my-design-system" => `--my-design-system-color-primary`
   */
  prefix?: string;

  /**
   * Terminal token path path which signifies that a token is also a group.
   *
   * Applies to all tokens in the project.
   * @see https://github.com/design-tokens/community-group/issues/97
   * @see https://github.com/amzn/style-dictionary/issues/716
   */
  groupMarkers?: string[];
}

export interface DTLSContext {
  /**
   * Documents manager that represents the state of all documents.
   */
  documents: Documents;
  /**
   * All tokens available to the server.
   */
  tokens: Tokens;
  /**
   * The Workspaces manager that represents the state of all workspaces.
   */
  workspaces: Workspaces;
  /**
   * The LSP server protocol implementation.
   */
  lsp?: Lsp;
}

export enum DTLSErrorCodes {
  /** The fallback value of a design token is incorrect. */
  incorrectFallback = "incorrect-fallback",
  /** The reference target does not appear to exist */
  unknownReference = "unknown-reference",
}

function requestMethodTypeGuard<M extends SupportedMethod>(
  method: M,
): RequestMethodTypeGuard<M> {
  return ((request) => request.method === method) as RequestMethodTypeGuard<M>;
}

const isCancelRequest = requestMethodTypeGuard("$/cancelRequest");
const isSetTraceRequest = requestMethodTypeGuard("$/setTrace");
const isInitializedRequest = requestMethodTypeGuard("initialized");
const isInitializeRequest = requestMethodTypeGuard("initialize");

/**
 * The Lsp class is responsible for processing LSP requests and notifications.
 * It handles the initialization of the server, and the processing of various LSP methods.
 */
export class Lsp {
  #handlers: Map<SupportedMethod, HandlerFor<SupportedMethod>>;
  #resolveInitialized!: () => void;
  #initialized = new Promise<void>((r) => this.#resolveInitialized = r);
  #initializationOptions?: LSP.LSPAny;
  #clientCapabilities?: LSP.ClientCapabilities;
  #traceLevel: LSP.TraceValues = LSP.TraceValues.Off;
  #cancelled = new Set<RequestId>();
  #documents: Documents;
  #workspaces: Workspaces;
  #tokens: Tokens;

  constructor(
    documents: Documents,
    workspaces: Workspaces,
    tokens: Tokens,
  ) {
    this.#documents = documents;
    this.#workspaces = workspaces;
    this.#tokens = tokens;
    this.#handlers = new Map(
      Object.entries({
        ...documents.handlers,
        ...workspaces.handlers,
        ...handlers,
      }) as [
        SupportedMethod,
        HandlerFor<SupportedMethod>,
      ][],
    );
  }

  get #context() {
    return {
      documents: this.#documents,
      tokens: this.#tokens,
      workspaces: this.#workspaces,
      lsp: this,
    };
  }

  #cancelRequest(request: LSP.RequestMessage) {
    const { id } = request;
    this.#cancelled.add(id);
    Logger.info`📵 Cancel ${id}`;
    return null;
  }

  #setTrace(value: LSP.TraceValues) {
    Logger.info`Set trace level to ${value}`;
    this.#traceLevel = value;
    return null;
  }

  /**
   * The initialize function is called when the server is initialized.
   * It registers the tokens files and sets up the server capabilities.
   *
   * @param params - The parameters for the initialization request.
   * @returns The capabilities of the server.
   */
  public async initialize(params: LSP.InitializeParams) {
    Logger.info`\n\n🎨 DESIGN TOKENS LANGUAGE SERVER 💎: ${
      params.clientInfo?.name ?? "unknown-client"
    }@${params.clientInfo?.version ?? "unknown-version"}\n`;

    try {
      const {
        capabilities,
        workspaceFolders,
        initializationOptions,
        trace,
      } = params;
      if (trace) this.#setTrace(trace);
      this.#clientCapabilities = capabilities;
      this.#initializationOptions = initializationOptions;
      if (workspaceFolders) {
        await this.#workspaces.add(this.#context, ...workspaceFolders);
      }
    } catch (error) {
      Logger.error`Failed to initialize the server: ${error}`;
    }

    return {
      capabilities: {
        textDocumentSync: LSP.TextDocumentSyncKind.Incremental,
        ...DocumentColor.capabilities,
        ...Hover.capabilities,
        ...Definition.capabilities,
        ...References.capabilities,
        ...Completion.capabilities,
        ...CodeAction.capabilities,
        ...Diagnostic.capabilities,
      },
      serverInfo: {
        name: "design-tokens-language-server",
        version,
      },
    } satisfies LSP.InitializeResult;
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
   * Processes the given request and returns the result.
   *
   * @param request - The request to process.
   * @returns The result of the request.
   */
  public async process(request: LSP.RequestMessage): Promise<unknown> {
    if (LSP.Message.isRequest(request) && this.#cancelled.has(request.id)) {
      return null;
    } else if (isInitializeRequest(request)) {
      return this.initialize(request.params);
    } else if (isInitializedRequest(request)) {
      await this.#workspaces.initialize(this.#context);
      return this.#resolveInitialized();
    } else if (isSetTraceRequest(request)) {
      return this.#setTrace(request.params.value);
    } else if (isCancelRequest(request)) {
      return this.#cancelRequest(request);
    } else if (request.method) {
      await this.#initialized;
      if (!this.#handlers.has(request.method as SupportedMethod)) {
        Logger.warn`❌ Unsupported method: ${request.method}`;
        return null;
      } else {
        const method = this.#handlers.get(request.method as SupportedMethod);
        return await method?.(
          request.params as SupportedParams,
          this.#context,
        ) ??
          null;
      }
    }
  }
}
