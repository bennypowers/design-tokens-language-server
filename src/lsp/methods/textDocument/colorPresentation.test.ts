import { beforeAll, describe, it } from "@std/testing/bdd";
import { expect } from "@std/expect";

import { colorPresentation } from "./colorPresentation.ts";
import { cssColorToLspColor } from "../../../css/color.ts";
import { register } from "#tokens";

describe("colorPresentation", () => {
  beforeAll(() => register({ path: "./test/tokens.json", prefix: "token" }));
  const uri = "file:///test.css";

  it("should return color presentations for matching colors", () => {
    const result = colorPresentation({
      textDocument: { uri },
      color: cssColorToLspColor("red"),
      range: {
        start: { line: 0, character: 0 },
        end: { line: 0, character: 0 },
      },
    });
    expect(result).toEqual([
      { label: "token-color-red" },
      { label: "token-color-red-hex" },
    ]);
  });
});
