import { calculatorExpressionItems } from "./calculator-items";

export function normalizeCalculatorInput(value: string): string {
  return value.replace(/[。０-９．＋－＊／（）]/g, char => char === "。" ? "." : String.fromCharCode(char.charCodeAt(0) - 0xfee0));
}

export function calculatorInputKey(key: string, code: string): string {
  return key === "Decimal" || (code === "NumpadDecimal" && key === ",") ? "." : normalizeCalculatorInput(key);
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

