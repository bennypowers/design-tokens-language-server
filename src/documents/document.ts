import * as LSP from "vscode-languageserver-protocol";
import type { DTLSContext } from "#lsp";
import { FullTextDocument } from "./textDocument.ts";

export abstract class DTLSTextDocument extends FullTextDocument {
  diagnostics: LSP.Diagnostic[] = [];
  abstract language: "json" | "css";
  abstract computeDiagnostics(_: DTLSContext): LSP.Diagnostic[];
}
