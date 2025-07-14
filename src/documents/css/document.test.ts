import { DTLSErrorCodes } from "#lsp/lsp.ts";
import { describe, it } from "@std/testing/bdd";
import { CssDocument, getLightDarkValues } from "#css";
import { expect } from "@std/expect/expect";
import { createTestContext } from "#test-helpers";

describe("getLightDarkValues", () => {
  it("should return light and dark values for a given value", () => {
    const value = "light-dark(red, maroon)";
    const [lightValue, darkValue] = getLightDarkValues(value);
    expect(lightValue).toBe("red");
    expect(darkValue).toBe("maroon");
  });

  it("should return an empty list for invalid value", () => {
    expect(getLightDarkValues("")).toEqual([]);
  });
});

const ctx = await createTestContext({
  testTokensSpecs: [
    {
      prefix: "token",
      spec: "/tokens.json",
      tokens: {
        color: {
          red: {
            _: {
              $value: "#ff0000",
              $type: "color",
            },
            hex: {
              $value: "#ff0000",
              $type: "color",
            },
          },
          "ld-color": {
            $value: "light-dark(hsl(0 100% 50%), hsl(240 100% 50%))",
            $type: "color",
          },
        },
        space: {
          small: {
            $value: "4px",
            $type: "size",
          },
        },
        font: {
          family: {
            "sans-serif": {
              $value: "Helvetica, Arial, sans-serif",
              $type: "fontFamily",
            },
          },
          weight: {
            thin: {
              $value: 100,
              $type: "fontWeight",
            },
          },
        },
      },
    },
  ],
});

describe("CssDocument", () => {
  it("should create a CssDocument instance", () => {
    const uri = "file:///test.css";
    const languageId = "css";
    const version = 1;
    const text = "body { color: red; }";

    const doc = CssDocument.create(ctx, uri, text, version);

    expect(doc.uri).toEqual(uri);
    expect(doc.languageId).toEqual(languageId);
    expect(doc.version).toEqual(version);
    expect(doc.getText()).toEqual(text);
    expect(doc.getFullRange()).toEqual({
      start: { line: 0, character: 0 },
      end: { line: 0, character: 20 },
    });
  });

  describe("diagnostics", () => {
    it("should not report incorrect fallback when semantically equivalent", () => {
      const text = `
        body {
          font-family: var(--token-font-family-sans-serif, Helvetica,Arial,sans-serif);
        }
      `;
      const doc = CssDocument.create(ctx, "file:///test.css", text);
      const diagnostics = doc.getDiagnostics(ctx);
      expect(diagnostics.length).toBe(0);
    });

    it("should report incorrect fallback", () => {
      const text = `
        body {
          font-family: var(--token-font-family-sans-serif, "Times New Roman");
        }
      `;
      const doc = CssDocument.create(ctx, "file:///test.css", text);
      const diagnostics = doc.getDiagnostics(ctx);
      expect(diagnostics.length).toBe(1);
      expect(diagnostics[0].code).toBe(DTLSErrorCodes.incorrectFallback);
    });

    it("should not report incorrect fallback for indented light-dark", () => {
      const text = `
        body {
          color: var(--token-color-ld-color, light-dark(
            hsl(0 100% 50%),
            hsl(240 100% 50%)
          ));
        }
      `;
      const doc = CssDocument.create(ctx, "file:///test.css", text);
      const diagnostics = doc.getDiagnostics(ctx);
      expect(diagnostics.length).toBe(0);
    });
  });
});
