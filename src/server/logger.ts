import { getFileSink } from "@logtape/file";
import { configure, getAnsiColorFormatter, getLogger } from "@logtape/logtape";

// const path = `${Deno.env.get("XDG_STATE_HOME") ?? `${Deno.env.get("HOME")}/.local/state`}/design-tokens-language-server/dtls.log`;
const path = `/var/home/bennyp/.local/state/design-tokens-language-server/dtls.log`;

await configure({
  sinks: {
    jsonc: getFileSink(path, {
      formatter: getAnsiColorFormatter({
        value: v => typeof v === 'string' ? v : Deno.inspect(v, {
          compact: true,
          breakLength: 100,
          colors: true,
          depth: 100,
          iterableLimit: 1000,
        })
      }),
    }),
  },
  loggers: [
    {
      category: 'dtls',
      lowestLevel: 'debug',
      sinks: ['jsonc'],
    }
  ]
});

export const Logger = getLogger('dtls');
