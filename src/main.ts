import { Server } from "#server";

import { parseArgs } from "@std/cli/parse-args";

import manifest from "../package.json" with { type: "json" };

const flags = parseArgs(Deno.args, {
  boolean: ["stdio", "socket"],
  string: ["version", "port"],
  default: { stdio: true },
  negatable: ["color"],
});

if (flags.version) {
  console.log("Design tokens language server");
  console.log(`version: ${manifest.version}`);
  Deno.exit(0);
}

if (flags.port && !flags.socket) {
  console.log("Design tokens language server");
  console.log("usage: --port=<port> --socket");
  console.log("usage: --stdio");
  Deno.exit(1);
}

if (flags.stdio) {
  Server.serve({ io: "stdio" });
} else if (flags.socket && flags.port) {
  const port = parseInt(flags.port);

  if (Number.isNaN(port)) {
    console.error("Invalid port number");
    Deno.exit(1);
  }

  // Server.serve({ io: 'socket', port });

  console.log(`Socket transport not implemented yet. Please use --stdio`);
  Deno.exit(1);
} else {
  console.log("Invalid arguments");
  Deno.exit(1);
}
