import type { CalculationValue } from "./mygo";

const operatorGlyphs: Record<string, string> = { "*": "×", "/": "÷", "-": "−" };

export function calculatorOperatorGlyph(token: string): string {
  return operatorGlyphs[token] ?? token;
}

export function isCalculatorMathGlyph(glyph: string): boolean {
  return /^[+−×÷()]$/.test(glyph);
}

export function formatCalculatorToken(token: string, language: string, grouping?: string): string {
  return /\d/.test(token) ? formatCalculation({ value: token, approximate: false }, language, grouping) : calculatorOperatorGlyph(token);
}

export function formatCalculation(result: CalculationValue, language: string, grouping = "auto"): string {
  const [integer = "", fraction] = result.value.split(".");
  const style = grouping === "auto" ? (language.startsWith("zh") ? "myriads" : "thousands") : grouping;
  const grouped = style === "myriads"
    ? integer.replace(/(\d)(?=\d{12}$)/, "$1兆").replace(/(\d)(?=\d{8}$)/, "$1亿").replace(/(\d)(?=\d{4}$)/, "$1万")
    : style === "thousands" ? integer.replace(/\B(?=(\d{3})+(?!\d))/g, ",") : integer;
  return (result.approximate ? "≈ " : "") + grouped + (fraction === undefined ? "" : `.${fraction}`);
}
