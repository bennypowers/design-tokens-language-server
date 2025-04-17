import type { Token } from "style-dictionary";

import { getLightDarkValues } from "./css/documents.ts";

function format(value: string): string {
  if (value?.startsWith?.("light-dark\(") && value.split("\n").length === 1) {
    const [light, dark] = getLightDarkValues(value);
    return `color: light-dark(
  ${light},
  ${dark}
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
