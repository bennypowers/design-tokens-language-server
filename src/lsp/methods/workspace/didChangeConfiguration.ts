import { DidChangeConfigurationParams } from "vscode-languageserver-protocol";
import { DTLSContextWithLsp } from "#lsp";
import { Logger } from "#logger";

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
export async function didChangeConfiguration(
  params: DidChangeConfigurationParams,
  context: DTLSContextWithLsp,
) {
  Logger.debug`User settings ${params.settings}`;
  await context.lsp.updateSettings(params.settings, context);
}
