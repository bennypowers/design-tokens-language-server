import type { Token } from "style-dictionary";

import { parse, stringify } from "./css/value-parser.ts";

function format(value: string): string {
  if (value?.startsWith?.("light-dark\(") && value.split("\n").length === 1) {
    const [light, , dark] = parse(value)?.pop()?.nodes ?? [];
    return `color: light-dark(
  ${stringify(light)},
  ${stringify(dark)}
)`;
  } else
    return value;
}

export function getTokenMarkdown(name: string, { $description, $value, $type }: Token) {
  return [
    `# \`--${name.replace(/^--/, '')}\``,
    '',
    // TODO: convert DTCG types to CSS syntax
    // const type = $type ? ` *<\`${$type}\`>*` : "";
    `Type: \`${$type}\``,
    $description ?? '',
    '',
    '```css',
    format($value),
    '```',
  ].join("\n");
}
