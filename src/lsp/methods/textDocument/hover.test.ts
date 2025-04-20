import { afterAll, beforeAll, describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { register, tokens } from "#tokens";
import { documents } from "#css";
import { hover } from "./hover.ts";

import { MarkupContent } from "vscode-languageserver-protocol";

describe("hover", () => {
  beforeAll(async () => {
    await register({ path: "./test/tokens.json", prefix: "token" });
  });

  afterAll(() => {
    tokens.clear();
  });

  it("should return hover information for a token", () => {
    documents.onDidOpen({
      textDocument: {
        uri: "file:///test.css",
        languageId: "css",
        version: 0,
        text: `a{b:var(--token-color-red)}\n`,
      },
    });

    const result = hover({
      textDocument: { uri: "file:///test.css" },
      position: { line: 0, character: 10 },
    });

    expect(result).not.toBeNull();
    expect(result?.contents).toHaveProperty("kind", "markdown");
    expect(result?.contents).toHaveProperty("value");
    expect((result?.contents as MarkupContent).value).toEqual(`\
      # \`--token-color-red\`

      Type: \`color\`
      Red colour

      \`\`\`css
      red
      \`\`\``.replaceAll(/^ {6}/gm, ""));
    expect(result?.range).toEqual({
      end: {
        character: 25,
        line: 0,
      },
      start: {
        character: 8,
        line: 0,
      },
    });
  });

  it("should return null for a non-token", () => {
    documents.onDidOpen({
      textDocument: {
        uri: "file:///test.css",
        languageId: "css",
        version: 0,
        text: `a{b:var(--non-token)}\n`,
      },
    });

    const result = hover({
      textDocument: { uri: "file:///test.css" },
      position: { line: 0, character: 10 },
    });

    expect(result).toBeNull();
  });

  it("should return formatted hover information for a token with a light-dark value", () => {
    documents.onDidOpen({
      textDocument: {
        uri: "file:///test.css",
        languageId: "css",
        version: 0,
        text: `a{b:var(--token-color-blue-lightdark)}`,
      },
    });

    const result = hover({
      textDocument: { uri: "file:///test.css" },
      position: { line: 0, character: 10 },
    });

    expect(result).not.toBeNull();
    expect(result?.contents).toHaveProperty("kind", "markdown");
    expect(result?.contents).toHaveProperty("value");
    expect((result?.contents as MarkupContent).value).toEqual(`\
      # \`--token-color-blue-lightdark\`

      Type: \`color\`
      Color scheme color

      \`\`\`css
      color: light-dark(
        lightblue,
        darkblue
      )
      \`\`\``.replaceAll(/^ {6}/gm, ""));
    expect(result?.range).toEqual({
      end: {
        character: 36,
        line: 0,
      },
      start: {
        character: 8,
        line: 0,
      },
    });
  });
});
