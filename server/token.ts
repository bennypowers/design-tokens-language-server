import type { Token } from "style-dictionary";

import { parse, stringify } from "./css/value-parser.ts";

export function getTokenMarkdown({ name, $description, $value, $type }: Token) {
  const desc = $description ? `: ${$description}` : "";
  const type = $type ? ` *<\`${$type}\`>*` : "";
  let value = $value;
  if ($value?.startsWith?.("light-dark\(") && $value.split("\n").length === 1) {
    const [light, , dark] = parse($value)?.pop()?.nodes ?? [];
    value = `light-dark(
  ${stringify(light)},
  ${stringify(dark)}
)`;
  }
  return [`\`--${name}\`${type}${desc}`, "", "```css", value, "```"].join("\n");
}
