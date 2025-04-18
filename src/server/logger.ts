import { getFileSink } from "@logtape/file";
import { configure, getAnsiColorFormatter, getLogger } from "@logtape/logtape";

const XDG_STATE_HOME = Deno.env.get("XDG_STATE_HOME") ??
  `${Deno.env.get("HOME")}/.local/state`;
const path = `${Deno.env.has('CI') ? Deno.cwd() : XDG_STATE_HOME}/design-tokens-language-server/dtls.log`;

const inspectValue = (v: unknown) =>
  typeof v === "string" ? v : Deno.inspect(v, {
    compact: true,
    breakLength: 100,
    colors: true,
    depth: 100,
    iterableLimit: 1000,
  });

await configure({
  sinks: {
    jsonc: getFileSink(path, {
      formatter: getAnsiColorFormatter({ value: inspectValue }),
    }),
  },
  loggers: [
    { category: ["logtape", "meta"], sinks: [] },
    {
      category: "dtls",
      lowestLevel: "debug",
      sinks: ["jsonc"],
    },
  ],
});

export const Logger = getLogger("dtls");
