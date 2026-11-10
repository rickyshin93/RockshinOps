import type { CalculationValue } from "./mygo";

export type CalculatorExpressionItem = {
  kind: "number" | "symbol" | "reference";
  value: string;
  offset: number;
};

export function calculatorExpressionItems(expression: string): CalculatorExpressionItem[] {
  return [...expression.matchAll(/#(?:[1-9][0-9]*|0)|[0-9]+(?:\.[0-9]*)?|\.[0-9]*|\S/g)].map(match => {
    const value = match[0];
    const kind = value.startsWith("#") && value.length > 1 ? "reference" : /^[0-9.]/.test(value) ? "number" : "symbol";
    return { kind, value, offset: match.index };
  });
}

export function calculatorReferenceBindings(expression: string, rows: readonly string[]): Record<string, string> {
  return Object.fromEntries((expression.match(/#[1-9][0-9]*/g) ?? []).map(token => [token, rows[Number(token.slice(1)) - 1] ?? ""]));
}

export function remapCalculatorReferences(expression: string, before: readonly string[], after: readonly string[]): string {
  if (!expression.includes("#") || before.every((id, index) => id === after[index])) return expression;
  const nextRow = new Map(after.map((id, index) => [id, index + 1]));
  return expression.replace(/#[1-9][0-9]*/g, token => {
    const id = before[Number(token.slice(1)) - 1];
    return id ? `#${nextRow.get(id) ?? 0}` : "#0";
  });
}

export function calculatorItemDeletion(expression: string, offset: number) {
  const items = calculatorExpressionItems(expression);
  const index = items.findIndex(item => item.offset === offset);
  const item = items[index];
  if (!item) return;
  let pair: CalculatorExpressionItem | undefined;
  if (item.value === "(" || item.value === ")") {
    const step = item.value === "(" ? 1 : -1;
    const opposite = step === 1 ? ")" : "(";
    let depth = 1;
    for (let i = index + step; ; i += step) {
      const candidate = items[i];
      if (!candidate) break;
      if (candidate.value === item.value) depth++;
      if (candidate.value === opposite && --depth === 0) { pair = candidate; break; }
    }
  }
  const removed = pair ? [item, pair].sort((a, b) => b.offset - a.offset) : [item];
  const next = removed.reduce((value, part) => value.slice(0, part.offset) + value.slice(part.offset + part.value.length), expression);
  return { expression: next, caret: Math.min(offset - (pair && pair.offset < offset ? 1 : 0), next.length) };
}

export function calculatorResultItems(result: CalculationValue) {
  return [
    { kind: "equality", value: result.approximate ? "≈" : "=" },
    { kind: "answer", value: result.value },
  ] as const;
}

export function calculatorItemAt(target: EventTarget | null) {
  return target instanceof Element ? target.closest<HTMLElement>("[data-calculation-item]") : null;
}
