import { beforeEach, describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { MarkupContent } from "vscode-languageserver-protocol";

import { createTestContext, DTLSTestContext } from "#test-helpers";

import { hover } from "./hover.ts";
import { JsonDocument } from "#json";

describe("textDocument/hover", () => {
  let ctx: DTLSTestContext;
  beforeEach(async () => {
    ctx = await createTestContext({
      testTokensSpecs: [
        {
          prefix: "token",
          spec: "file:///tokens.json",
          tokens: {
            color: {
              $type: "color",
              red: {
                _: {
                  $value: "#ff0000",
                  $description: "Red colour",
                },
                hex: {
                  $value: "{color.red._}",
                  $description: "Red colour (by reference)",
                },
              },
              blue: {
                lightdark: {
                  $value: "light-dark(lightblue, darkblue)",
                  $description: "Color scheme color",
                },
              },
            },
            space: {
              $type: "size",
              small: {
                $value: "4px",
              },
            },
          },
        },
        {
          prefix: "token",
          spec: "file:///referer.json",
          tokens: {
            color: {
              $type: "color",
              hexref: {
                $value: "{color.red.hex}",
              },
            },
          },
        },
      ],
    });
  });

  it("should return hover information for a token", () => {
    const textDocument = ctx.documents.createDocument(
      "css",
      /*css*/ `
      a {
        color:var(--token-color-red);
      }
    `,
    );
    const doc = ctx.documents.get(textDocument.uri);
    const position = doc.positionForSubstring("--token-color-red");

    const result = hover({ textDocument, position }, ctx);

    expect(result).not.toBeNull();
    expect(result?.range).toEqual(
      doc.getRangeForSubstring("--token-color-red"),
    );
    expect(result?.contents).toHaveProperty("kind", "markdown");
    expect(result?.contents).toHaveProperty("value");
    expect((result?.contents as MarkupContent).value).toEqual(`\
      # \`--token-color-red\`

      Type: \`color\`
      Red colour

      \`\`\`css
      #ff0000
      \`\`\``.replaceAll(/^ {6}/gm, ""));
  });

  it("should return null for a non-token", () => {
    const textDocument = ctx.documents.createDocument(
      "css",
      /*css*/ `
      a {
        color: :var(--non-token);
      }
    `,
    );
    const doc = ctx.documents.get(textDocument.uri);
    const position = doc.positionForSubstring("--non-token");
    const result = hover({ textDocument, position }, ctx);
    expect(result).toBeNull();
  });

  it("should return formatted hover information for a token with a light-dark value", () => {
    const textDocument = ctx.documents.createDocument(
      "css",
      /*css*/ `
      a{
        color: var(--token-color-blue-lightdark);
      }
    `,
    );

    const doc = ctx.documents.get(textDocument.uri);
    const position = doc.positionForSubstring("--token-color-blue-lightdark");
    const range = doc.getRangeForSubstring("--token-color-blue-lightdark");
    const result = hover({ textDocument, position }, ctx);
    expect(result).not.toBeNull();
    expect(result?.range).toEqual(range);
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
  });

  describe("in a json file with local references", () => {
    it("should return hover information for a token", () => {
      const doc = ctx.documents.get("file:///tokens.json") as JsonDocument;
      const range = doc.getRangeForSubstring("color.red._");
      const position = range.start;
      const result = hover({ textDocument: doc, position }, ctx);
      expect(result).toEqual({
        range,
        contents: {
          kind: "markdown",
          value: `# \`color.red._\`

            Type: \`color\`
            Red colour

            \`\`\`css
            #ff0000
            \`\`\``.replace(/^ {12}/gm, ""),
        },
      });
    });
  });

  describe("in a json file with foreign references", () => {
    it("should return hover information for a token", () => {
      const doc = ctx.documents.get("file:///referer.json") as JsonDocument;
      const range = doc.getRangeForSubstring("color.red.hex");
      const position = range.start;
      const result = hover({ textDocument: doc, position }, ctx);
      expect(result).toEqual({
        range,
        contents: {
          kind: "markdown",
          value: `# \`color.red.hex\`

            Type: \`color\`
            Red colour (by reference)

            \`\`\`css
            #ff0000
            \`\`\``.replace(/^ {12}/gm, ""),
        },
      });
    });
  });
});
