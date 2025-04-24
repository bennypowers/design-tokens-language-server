import { describe, it } from "@std/testing/bdd";
import { getLightDarkValues } from "#css";
import { expect } from "@std/expect/expect";

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
