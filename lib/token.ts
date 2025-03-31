import { Token } from "style-dictionary";

export function getTokenMarkdown(token: Token) {
  return [
    `\`--${token.name}\` *<\`${token.$type}\`>*:`,
    `  **${token.$value}**`,
    token.$description,
    token.comment,
  ].filter(Boolean).join('\n\n')
}
