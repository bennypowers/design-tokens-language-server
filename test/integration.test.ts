// deno-lint-ignore-file no-explicit-any
import * as LSP from "vscode-languageserver-protocol";
import { afterAll, beforeAll, describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";
import { createTestLspClient, getLspRangesForSubstring } from "#test-helpers";

import manifest from "../package.json" with { type: "json" };

const { version } = manifest;

const rootUrl = new URL("../test/package/", import.meta.url);
const rootUri = rootUrl.href;

const testCssUrl = new URL("./package/test.css", import.meta.url);

const refererYamlUrl = new URL(
  "./package/tokens/referer.yaml",
  import.meta.url,
);

const refererJsonUrl = new URL(
  "./package/tokens/referer.json",
  import.meta.url,
);

const refereeJsonUrl = new URL(
  "./package/tokens/referee.json",
  import.meta.url,
);

const refererJsonContent = await Deno.readTextFile(refererJsonUrl);
const refereeJsonContent = await Deno.readTextFile(refereeJsonUrl);
const testCssContent = await Deno.readTextFile(testCssUrl);

describe("design-tokens-language-server", () => {
  let client: ReturnType<typeof createTestLspClient>;

  beforeAll(() => {
    client = createTestLspClient();
  });

  afterAll(async () => {
    await client.close();
  });

  describe("initialize", () => {
    let initializeResponse: any;

    beforeAll(async () => {
      initializeResponse = await client.sendMessage({
        method: "initialize",
        params: {
          processId: null,
          rootUri,
          workspaceFolders: [{ uri: rootUri, name: "root" }],
          clientInfo: {
            name: "DENO_TEST_CLIENT",
            version: Temporal.Now.plainDateTimeISO().toString(),
          },
          capabilities: {
            textDocument: {
              synchronization: {
                dynamicRegistration: false,
                willSave: false,
                didSave: false,
                willSaveWaitUntil: false,
              },
            },
          },
        },
      });
    });

    it("should initialize the LSP server", () => {
      expect(initializeResponse?.jsonrpc).toBe("2.0");
      expect(initializeResponse?.id).toBe(0);
      expect(initializeResponse?.result.serverInfo.version).toBe(version);
      expect(initializeResponse?.result.serverInfo.name).toBe(
        "design-tokens-language-server",
      );
    });

    describe("initialized", () => {
      beforeAll(async () => {
        await client.sendNotification({ method: "initialized" });
      });

      describe("didOpen test.css", () => {
        let didOpenResponse: any;

        beforeAll(async () => {
          didOpenResponse = await client.sendNotification({
            method: "textDocument/didOpen",
            params: {
              textDocument: {
                uri: testCssUrl.href,
                languageId: "css",
                version: 1,
                text: testCssContent,
              },
            },
          });
        });

        it("should not respond to the didOpen notification", () => {
          // Step 3: Open a document
          expect(didOpenResponse).toBeUndefined(); // No response expected for didOpen
        });

        describe("didChange test.css", () => {
          // Step 4: Simulate incremental document changes
          beforeAll(async () => {
            const [originalRange] = getLspRangesForSubstring(
              testCssContent,
              ", green",
            );
            // First incremental update: Change ", green" to ", breen" in test.css
            await client.sendNotification({
              method: "textDocument/didChange",
              params: {
                textDocument: { uri: testCssUrl.href, version: 2 },
                contentChanges: [
                  {
                    text: ", breen",
                    range: originalRange,
                  },
                ],
              },
            });

            // Second incremental update: Change ", breen" back to ", green"
            await client.sendNotification({
              method: "textDocument/didChange",
              params: {
                textDocument: { uri: testCssUrl.href, version: 3 },
                contentChanges: [
                  {
                    text: ", green",
                    range: originalRange,
                  },
                ],
              },
            });
          });

          describe("textDocument/hover", () => {
            let hoverResponse: any;

            describe("on non-token", () => {
              beforeAll(async () => {
                // Step 5: Request hover and diagnostics
                hoverResponse = await client.sendMessage({
                  method: "textDocument/hover",
                  params: {
                    textDocument: { uri: testCssUrl.href },
                    position: { line: 0, character: 0 },
                  },
                });
              });

              it("should return null hover information", () => {
                expect(hoverResponse.result).toBeNull();
              });
            });

            describe("on token name", () => {
              const tokenName = "--token-color-blue-lightdark";
              let range: LSP.Range;
              beforeAll(async () => {
                [range] = getLspRangesForSubstring(testCssContent, tokenName);
                const position = range!.start!;
                // Step 5: Request hover and diagnostics
                hoverResponse = await client.sendMessage({
                  method: "textDocument/hover",
                  params: {
                    textDocument: { uri: testCssUrl.href },
                    position,
                  },
                });
              });

              it("should return hover information", () => {
                // Step 6: Assert results
                expect(hoverResponse.result).toEqual({
                  range,
                  contents: {
                    kind: "markdown",
                    value: `# \`${tokenName}\`

Type: \`color\`
Color scheme color

\`\`\`css
color: light-dark(
  lightblue,
  darkblue
)
\`\`\``,
                  },
                });
              });
            });
          });

          describe("textDocument/diagnostic", () => {
            let diagnosticsResponse: any;

            beforeAll(async () => {
              diagnosticsResponse = await client.sendMessage({
                method: "textDocument/diagnostic",
                params: { textDocument: { uri: testCssUrl.href } },
              });
            });
            it("returns full diagnostics", () => {
              expect(diagnosticsResponse.result.kind).toEqual("full");
            });
            it("returns 4 diagnostics", () => {
              expect(diagnosticsResponse.result.items).toHaveLength(4);
            });
            it("returns the right diagnostics", () => {
              const [$1, $2, $3, $4] = diagnosticsResponse.result.items;
              expect($1, "1st diagnostic").toEqual({
                severity: 1,
                code: "incorrect-fallback",
                message: "Token fallback does not match expected value: blue",
                data: {
                  tokenName: "--token-color-blue",
                  actual: "green",
                  expected: "blue",
                },
                range: getLspRangesForSubstring(testCssContent, "green").at(2),
              });
              expect($2, "2nd diagnostic").toEqual({
                severity: 1,
                code: "incorrect-fallback",
                message:
                  "Token fallback does not match expected value: 'Super Duper', Helvetica, Arial, sans-serif",
                data: {
                  tokenName: "--token-font-family",
                  expected: "'Super Duper', Helvetica, Arial, sans-serif",
                  actual: "fee, fi, fo, fum",
                },
                range: getLspRangesForSubstring(testCssContent, "fee").at(0),
              });
              expect($3, "3rd diagnostic").toEqual({
                severity: 1,
                code: "incorrect-fallback",
                data: {
                  tokenName: "--token-font-weight",
                  actual: '"400"',
                  expected: 400,
                },
                message: "Token fallback does not match expected value: 400",
                range: getLspRangesForSubstring(testCssContent, '"400"').at(0),
              });
              expect($4, "4th diagnostic").toEqual({
                severity: 1,
                code: "incorrect-fallback",
                message:
                  "Token fallback does not match expected value: 1px 2px 3px 4px rgba(2, 4, 6 / 0.8)",
                data: {
                  tokenName: "--token-box-shadow",
                  actual: "1px 2px 3px 4px rgba(2, 4, 6 / .8)",
                  expected: "1px 2px 3px 4px rgba(2, 4, 6 / 0.8)",
                },
                range: getLspRangesForSubstring(
                  testCssContent,
                  "1px 2px 3px 4px rgba(2, 4, 6 / .8)",
                ).at(0),
              });
            });
          });

          describe("didOpen referer.yaml", () => {
            let refererYamlContent: string;

            beforeAll(async () => {
              refererYamlContent = await Deno.readTextFile(refererYamlUrl);

              // Step 7: Open YAML referer
              await client.sendNotification({
                method: "textDocument/didOpen",
                params: {
                  textDocument: {
                    uri: refererYamlUrl.href,
                    languageId: "yaml",
                    version: 1,
                    text: refererYamlContent,
                  },
                },
              });
            });

            describe("textDocument/references", () => {
              let referencesResponse: any;

              beforeAll(async () => {
                referencesResponse = await client.sendMessage({
                  method: "textDocument/references",
                  params: {
                    textDocument: {
                      uri: refererYamlUrl.href,
                    },
                    context: {
                      includeDeclaration: true,
                    },
                    position: getLspRangesForSubstring(
                      refererYamlContent,
                      "color.red.hex",
                    ).at(0)!.start!,
                  },
                });
              });

              it("gathers references", () => {
                const refererJsonColorRedHexRanges = getLspRangesForSubstring(
                  refererJsonContent,
                  `{color.red.hex}`,
                );

                const refererYamlColorRedHexRanges = getLspRangesForSubstring(
                  refererYamlContent,
                  "{color.red.hex}",
                );

                // TODO: get context from the test client and compute these values
                expect(referencesResponse.result).toEqual([
                  ...refererJsonColorRedHexRanges.map((range) => ({
                    uri: refererJsonUrl.href,
                    range,
                  })),
                  ...refererYamlColorRedHexRanges.map((range) => ({
                    uri: refererYamlUrl.href,
                    range,
                  })),
                  {
                    uri: testCssUrl.href,
                    range: getLspRangesForSubstring(
                      testCssContent,
                      "--token-color-red-hex",
                    ).at(0), // the second occurrence is `--token-color-red-hexref`
                  },
                  {
                    uri: refereeJsonUrl.href,
                    range: getLspRangesForSubstring(
                      refereeJsonContent,
                      `{
        "$value": "#F00",
        "$description": "Red colour (hex)"
      }`,
                    ).at(0),
                  },
                ]);
              });
            });
          });
        });
      });
    });
  });
});
