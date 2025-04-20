import { Server } from "./server.ts";
import { Lsp } from "./lsp/lsp.ts";

Server.serve(Lsp);
