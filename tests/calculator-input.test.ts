import { expect, test } from "bun:test";
import { calculatorInputKey, isCalculatorDraft, normalizeCalculatorInput, normalizeCalculatorZeros, queueCalculatorInput } from "../src/calculator-input";
import { calculatorExpressionItems, calculatorItemDeletion, calculatorResultItems, remapCalculatorReferences } from "../src/calculator-items";

test("calculator items keep expression offsets and separate the sign from the answer", () => {
  expect(calculatorExpressionItems("1.5×(#12-0.5)")).toEqual([
    { kind: "number", value: "1.5", offset: 0 },
    { kind: "symbol", value: "×", offset: 3 },
    { kind: "symbol", value: "(", offset: 4 },
    { kind: "reference", value: "#12", offset: 5 },
    { kind: "symbol", value: "-", offset: 8 },
    { kind: "number", value: "0.5", offset: 9 },
    { kind: "symbol", value: ")", offset: 12 },
  ]);
  expect(calculatorResultItems({ value: "0.3333", approximate: true })).toEqual([
    { kind: "equality", value: "≈" },
    { kind: "answer", value: "0.3333" },
  ]);
});

test("calculator draft accepts supported arithmetic and unfinished prefixes", () => {
  for (const value of ["", "-.", "2+", "2×(", "2×-3", "(-2+.5)", "2+#1", "#1×#2"]) {
    expect(isCalculatorDraft(value), value).toBe(true);
  }
  expect(normalizeCalculatorInput("２＋（３．５）")).toBe("2+(3.5)");
});

test("deleting an item removes exactly the token or its matching parenthesis pair", () => {
  expect(calculatorItemDeletion("2×(3+(4))", 2)).toEqual({ expression: "2×3+(4)", caret: 2 });
  expect(calculatorItemDeletion("2×(3+(4))", 7)).toEqual({ expression: "2×(3+4)", caret: 6 });
  expect(calculatorItemDeletion("2+#12", 2)).toEqual({ expression: "2+", caret: 2 });
});

test("row deletion keeps surviving draft references and marks the removed source unavailable", () => {
  const rows = Array.from({ length: 10 }, (_, index) => `id${index + 1}`);
  const remapped = remapCalculatorReferences("#1+#2+#3+#10+#0", rows, rows.filter(id => id !== "id2"));
  expect(remapped).toBe("#1+#0+#2+#9+#0");
  expect(isCalculatorDraft(remapped)).toBe(true);
  expect(calculatorExpressionItems(remapped)[2]).toEqual({ kind: "reference", value: "#0", offset: 3 });
});

test("draft references survive multiple changes and unrelated appends by row identity", () => {
  const rows = ["a", "b", "c", "d"];
  expect(remapCalculatorReferences("#2+#4", rows, ["b", "d", "e"])).toBe("#1+#2");
  expect(remapCalculatorReferences("#2+#4", rows, [...rows, "e"])).toBe("#2+#4");
});
test("decimal input normalizes Chinese punctuation and the numeric keypad without accepting a plain comma", () => {
  expect(normalizeCalculatorInput("１。５+２．２５")).toBe("1.5+2.25");
  expect(calculatorInputKey(",", "NumpadDecimal")).toBe(".");
  expect(calculatorInputKey(",", "Comma")).toBe(",");
  expect(calculatorInputKey("Delete", "NumpadDecimal")).toBe("Delete");
});

test("calculator draft rejects non-arithmetic text and adjacent operands", () => {
  for (const value of ["1+hi", "foo", "2&3", "1=1", "2📱", "1..2", "1+×", "1 2", "2+()", "1(#2)", "1#2", "#1#2", "#1 5", "1(2)", ".+", "1)", "2 + 3", " 2", "2\u3000+3", "2\n+3"]) {
    expect(isCalculatorDraft(value), value).toBe(false);
  }
});

test("leading zeroes are removed from integer operands without changing decimal places or references", () => {
  for (const [value, expected, caret] of [
    ["99-00009", "99-9", 4], ["00", "0", 1], ["000.0050", "0.0050", 6],
    ["#100+0002", "#100+2", 6], ["10+0.00", "10+0.00", 7],
  ] as const) expect(normalizeCalculatorZeros(value, value.length)).toEqual({ value: expected, caret });
  expect(normalizeCalculatorZeros("0009", 2)).toEqual({ value: "9", caret: 0 });
});

test("input while saving can build a new expression or continue the answer", () => {
  let value = "";
  for (const key of ["９", "+", "0", "2", "Backspace", "5"]) value = queueCalculatorInput(value, key)!;
  expect(value).toBe("9+5");
  for (const [input, want] of [["+", "#1+"], ["(", "#1×("], [".", "."], ["#2", null], [")", null]]) {
    expect(queueCalculatorInput("", input!)).toBe(want);
  }
  expect(queueCalculatorInput("#1+", "Backspace")).toBe("");
  expect(queueCalculatorInput("#1+3", "AC")).toBe("");
  expect(queueCalculatorInput("", "+002", false)).toBe("+2");
  expect(queueCalculatorInput("#1+", "2*3", false)).toBe("#1+2*3");
  let grouped = "";
  for (const key of ["2", "(", "3", ")"]) grouped = queueCalculatorInput(grouped, key)!;
  expect(grouped).toBe("2×(3)");
});

test("deleting an operator never retargets the preceding reference", () => {
  expect(calculatorItemDeletion("#1+12", 2)).toBeNull();
  expect(calculatorItemDeletion("#19+3", 3)).toBeNull();
  expect(calculatorItemDeletion("12+#1", 3)).toEqual({ expression: "12+", caret: 3 });
});
