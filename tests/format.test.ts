import { describe, expect, test } from "bun:test";
import { capacityParts, formatCapacity } from "../src/format";

describe("decimal capacity display", () => {
  test.each([
    [0, "0.0 GB"], [500_000_000, "500.0 MB"], [1_000_000_000, "1.0 GB"],
    [90_671_276_032, "90.7 GB"], [1_234_000_000_000, "1.23 TB"],
  ])("%d bytes → %s", (bytes, expected) => {
    expect(formatCapacity(bytes as number)).toBe(expected);
  });
  test("provides separate numeric and unit text for the large readout", () => {
    expect(capacityParts(90_671_276_032)).toEqual({ value: "90.7", unit: "GB" });
  });
});
