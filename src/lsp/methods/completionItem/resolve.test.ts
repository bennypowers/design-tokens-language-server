import { describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { createTestContext } from "#test-helpers";

import { resolve } from "./resolve.ts";

describe("completionItem/resolve", () => {
  const ctx = createTestContext({
    testTokensSpecs: [
      {
        prefix: "token",
        spec: "file:///tokens.json",
        tokens: {
          color: {
            red: {
              $value: "#ff0000",
              $type: "color",
              $description: "A red color",
            },
          },
        },
      },
    ],
  });

  describe("given a completion item with a token", () => {
    const completionItem = {
      label: "--token-color-red",
    };
    it("should resolve the completion item with details and documentation", () => {
      const resolvedItem = resolve(completionItem, ctx);
      expect(resolvedItem).toEqual({
        ...completionItem,
        labelDetails: {
          detail: ": #ff0000",
        },
        documentation: {
          kind: "markdown",
          value:
            "# `--token-color-red`\n\nType: `color`\nA red color\n\n```css\n#ff0000\n```",
        },
      });
    });
  });
});
