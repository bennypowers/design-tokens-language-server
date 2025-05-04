import { getFileSink } from "@logtape/file";
import { configure, getAnsiColorFormatter, getLogger } from "@logtape/logtape";
import { join } from "@std/path";

const XDG_STATE_HOME = Deno.env.get("XDG_STATE_HOME");
const HOME = Deno.env.get("HOME");
const isCi = Deno.env.has("CI");

const serverName = "design-tokens-language-server";

const logDir = isCi
  ? Deno.cwd()
  : XDG_STATE_HOME
  ? join(XDG_STATE_HOME, serverName)
  : HOME
  ? join(HOME, ".local", "state", serverName)
  : join(Deno.cwd(), ".dtls-log");

const path = join(logDir, "dtls.log");

await Deno.mkdir(logDir, { recursive: true });

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
    file: getFileSink(path, {
      formatter: getAnsiColorFormatter({ value: inspectValue }),
    }),
  },
  loggers: [
    { category: ["logtape", "meta"], sinks: [] },
    {
      category: "dtls",
      lowestLevel: "debug",
      sinks: ["file"],
    },
  ],
});

export const Logger = getLogger("dtls");

export function logged(tag?: string) {
  return function loggedMethod<This, Args extends any[], Return>(
    target: (this: This, ...args: Args) => Return,
    context: ClassMethodDecoratorContext<
      This,
      (this: This, ...args: Args) => Return
    >,
  ) {
    const methodName = String(context.name);
    return function replacementMethod(this: This, ...args: Args): Return {
      Logger.debug`${tag ?? ""}${target.name}.${methodName}(...${args})`;
      const result = target.call(this, ...args);
      Logger.debug`${tag ?? ""}${target.name}.${methodName}: ${result}`;
      return result;
    };
  };
}
