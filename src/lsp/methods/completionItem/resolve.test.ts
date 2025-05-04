import { beforeEach, describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { createTestContext, DTLSTestContext } from "#test-helpers";

import { resolve } from "./resolve.ts";

describe("completionItem/resolve", () => {
  let ctx: DTLSTestContext;

  beforeEach(async () => {
    ctx = await createTestContext({
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
              blue: {
                $value: "#0000ff",
                $type: "color",
              },
            },
          },
        },
      ],
    });
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

  describe("given a completion item with a token that has no description", () => {
    const completionItem = {
      label: "--token-color-blue",
    };
    it("should resolve the completion item with details and documentation", () => {
      const resolvedItem = resolve(completionItem, ctx);
      expect(resolvedItem).toEqual({
        ...completionItem,
        labelDetails: {
          detail: ": #0000ff",
        },
        documentation: {
          kind: "markdown",
          value:
            "# `--token-color-blue`\n\nType: `color`\n\n```css\n#0000ff\n```",
        },
      });
    });
  });

  describe("given a completion item that somehow represents a non-token", () => {
    const completionItem = {
      label: "--token-bogus",
    };
    it("should resolve the completion item with details and documentation", () => {
      const resolvedItem = resolve(completionItem, ctx);
      expect(resolvedItem).toEqual(completionItem);
    });
  });
});
