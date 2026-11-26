import { calculatorExpressionItems } from "./calculator-items";

export function normalizeCalculatorInput(value: string): string {
  return value.replace(/[。０-９．＋－＊／（）]/g, char => char === "。" ? "." : String.fromCharCode(char.charCodeAt(0) - 0xfee0));
}

// Normalize integer operands without touching fractional digits or reference IDs.
export function normalizeCalculatorZeros(value: string, caret: number): { value: string; caret: number } {
  let removed = 0;
  let removedBeforeCaret = 0;
  for (const item of calculatorExpressionItems(value)) {
    if (item.kind !== "number") continue;
    const count = item.value.match(/^0+(?=\d)/)?.[0].length ?? 0;
    if (!count) continue;
    const start = item.offset - removed;
    value = value.slice(0, start) + value.slice(start + count);
    removedBeforeCaret += Math.max(0, Math.min(count, caret - item.offset));
    removed += count;
  }
  return { value, caret: caret - removedBeforeCaret };
}

export function calculatorInputKey(key: string, code: string): string {
  return key === "Decimal" || (code === "NumpadDecimal" && key === ",") ? "." : normalizeCalculatorInput(key);
}

// #1 is a placeholder for the answer being saved. Pasted text cannot supply references.
export function queueCalculatorInput(current: string, input: string, fromKey = true): string | null {
  if (fromKey && input === "AC") return "";
  if (fromKey && input === "Backspace") return current === "#1" || current.length === 3 && current.startsWith("#1") ? "" : current.slice(0, -1);
  const text = normalizeCalculatorInput(input);
  if (text.includes("#")) return null;
  let expression = !current && fromKey && /^[+−×÷*/(-]$/.test(text)
    ? `#1${text === "(" ? "×(" : text}` : current + text;
  if (fromKey && text === "(" && !isCalculatorDraft(expression)) expression = current + "×(";
  return isCalculatorDraft(expression) ? normalizeCalculatorZeros(expression, expression.length).value : null;
}

// Unlike the evaluator, the editor allows a prefix, such as "2 × (".
export function isCalculatorDraft(value: string): boolean {
  if (/\s/.test(value)) return false;
  const items = calculatorExpressionItems(value);
  let operand = true;
  let signed = false;
  let depth = 0;
  for (const [index, { kind, value: token }] of items.entries()) {
    if (token === "(") {
      if (!operand) return false;
      depth++;
      signed = false;
    } else if (token === ")") {
      if (operand || depth === 0) return false;
      depth--;
    } else if (/^[+−×÷*/-]$/.test(token)) {
      if (operand && (signed || !"+−-".includes(token))) return false;
      signed = operand;
      operand = true;
    } else {
      if (!operand || kind === "symbol" || (token === "." && index !== items.length - 1)) return false;
      operand = false;
    }
  }
  return true;
}

