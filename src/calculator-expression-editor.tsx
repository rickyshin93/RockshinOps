import { useImperativeHandle, useLayoutEffect, useRef, type KeyboardEvent, type Ref } from "react";
import { calculatorInputKey, isCalculatorDraft, normalizeCalculatorInput } from "./calculator-input";
import { calculatorOperatorGlyph, isCalculatorMathGlyph } from "./calculation-format";
import { calculatorExpressionItems, calculatorItemAt, calculatorItemDeletion } from "./calculator-items";

export type LinkedOperand = { text: string; title: string; value?: string };
export type CalculatorCaretPosition = { offset: number; preferNext?: boolean };
export type CalculatorEditor = {
  focus: (options?: FocusOptions) => void;
  focusEnd: () => void;
  focusToken: (offset: number, position: CalculatorCaretPosition) => void;
  scrollIntoView: (options: ScrollIntoViewOptions) => void;
  insert: (text: string, backspace?: boolean) => boolean;
  replayBackspaces: (count: number) => number;
  removeItem: () => boolean;
};

const referenceAtEnd = /#(?:[1-9][0-9]*|0)$/;
const numericInput = /^[0-9.]+$/;
const groupingMarks = ",万亿兆";

export function calculatorTokenPosition(token: HTMLElement, x: number, y: number): CalculatorCaretPosition {
  const walker = document.createTreeWalker(token, NodeFilter.SHOW_TEXT);
  const range = document.createRange();
  let offset = 0;
  let position: CalculatorCaretPosition = { offset: Infinity };
  let nearest = Infinity;
  for (let text = walker.nextNode() as Text | null; text; text = walker.nextNode() as Text | null) {
    for (let i = 0; i < text.length; i++) {
      range.setStart(text, i);
      range.setEnd(text, i + 1);
      const { left, right, top, bottom } = range.getBoundingClientRect();
      const vertical = Math.max(top - y, y - bottom, 0);
      const grouping = groupingMarks.includes(text.data.charAt(i));
      const next = grouping ? offset : offset + 1;
      for (const [edge, candidate, preferNext] of [[left, offset, !grouping], [right, next, grouping]] as const) {
        const distance = Math.hypot(edge - x, vertical);
        if (distance < nearest) { nearest = distance; position = { offset: candidate, preferNext }; }
      }
      offset = next;
    }
  }
  return position;
}

function withoutGrouping(text: string) {
  return text.replace(/[,万亿兆]/g, "");
}

function referenceSelection(field: HTMLElement) {
  const chip = field.querySelector<HTMLElement>("[data-calculation-selected][data-calculation-link]");
  const selection = document.getSelection();
  const range = selection?.rangeCount ? selection.getRangeAt(0) : null;
  if (!chip || !range || !chip.contains(range.commonAncestorContainer)) return null;
  const prefix = document.createRange();
  prefix.selectNodeContents(chip);
  prefix.setEnd(range.startContainer, range.startOffset);
  const from = withoutGrouping(prefix.toString()).length;
  prefix.setEnd(range.endContainer, range.endOffset);
  return { chip, token: chip.dataset.calculationLink!, offset: Number(chip.dataset.calculationToken), from, to: withoutGrouping(prefix.toString()).length };
}

function raw(node: Node): string {
  if (node instanceof HTMLElement && node.dataset.calculationLink) return node.dataset.calculationLink;
  if (node instanceof HTMLElement && node.dataset.calculationOperator &&
      node.textContent === calculatorOperatorGlyph(node.dataset.calculationOperator)) return node.dataset.calculationOperator;
  if (node.nodeType === Node.TEXT_NODE) return node.textContent ?? "";
  let value = "";
  let linked = false;
  for (const child of node.childNodes) {
    const text = raw(child);
    if (!text) continue;
    // A digit entered next to an existing chip must never change its target.
    if (linked && /^[0-9０-９]/.test(text)) value += " ";
    value += text;
    linked = child instanceof HTMLElement && !!child.dataset.calculationLink;
  }
  return value;
}

function hasTypedReference(node: Node): boolean {
  if (node instanceof HTMLElement && node.dataset.calculationLink) return false;
  if (node.nodeType === Node.TEXT_NODE) return !!node.textContent?.includes("#");
  return [...node.childNodes].some(hasTypedReference);
}

function selectionOffsets(field: HTMLElement): [number, number] {
  const selection = document.getSelection();
  const range = selection?.rangeCount ? selection.getRangeAt(0) : null;
  if (!range || !field.contains(range.commonAncestorContainer)) {
    const end = raw(field).length;
    return [end, end];
  }
  const before = document.createRange();
  before.selectNodeContents(field);
  before.setEnd(range.startContainer, range.startOffset);
  const start = raw(before.cloneContents()).length;
  before.setEnd(range.endContainer, range.endOffset);
  return [start, raw(before.cloneContents()).length];
}

function textPosition(field: HTMLElement, offset: number, preferNext = false): [Node, number] {
  for (const [index, node] of Array.from(field.childNodes).entries()) {
    const length = raw(node).length;
    if (preferNext && offset === length && node.nextSibling) {
      offset -= length;
      continue;
    }
    if (offset <= length) {
      if (node.nodeType === Node.TEXT_NODE) return [node, offset];
      if (node instanceof HTMLElement && !node.dataset.calculationLink && node.firstChild) return textPosition(node, offset, preferNext);
      return [field, index + (offset > 0 ? 1 : 0)];
    }
    offset -= length;
  }
  return [field, field.childNodes.length];
}

function placeCaret(node: Node, offset: number) {
  const range = document.createRange();
  range.setStart(node, offset);
  range.collapse(true);
  selectRange(range);
}

function selectRange(range: Range) {
  const selection = document.getSelection();
  selection?.removeAllRanges();
  selection?.addRange(range);
}

function syncCaretEdge(field: HTMLElement) {
  const token = field.querySelector<HTMLElement>("[data-calculation-selected]");
  if (!token) return;
  const selection = document.getSelection();
  let edge: string | undefined;
  if (selection?.isCollapsed && selection.anchorNode && token.contains(selection.anchorNode)) {
    const prefix = document.createRange();
    prefix.selectNodeContents(token);
    prefix.setEnd(selection.anchorNode, selection.anchorOffset);
    const length = prefix.toString().length;
    edge = length === 0 ? "start" : length === token.textContent?.length ? "end" : undefined;
  }
  // WebKit can paint its native caret against the neighboring token at text boundaries.
  if (edge !== token.dataset.calculationCaretEdge) {
    if (edge) token.dataset.calculationCaretEdge = edge;
    else token.removeAttribute("data-calculation-caret-edge");
  }
}

function paint(field: HTMLElement, value: string, operand: (token: string) => LinkedOperand, display: (token: string) => string) {
  const nodes: Node[] = [];
  let start = 0;
  for (const { value: token, offset: index, kind } of calculatorExpressionItems(value)) {
    if (index > start) nodes.push(document.createTextNode(value.slice(start, index)));
    const part = document.createElement("span");
    part.dataset.calculationItem = kind;
    part.dataset.calculationToken = String(index);
    if (kind === "reference") {
      part.className = "calculator-reference";
      part.contentEditable = "false";
      part.dataset.calculationLink = token;
      part.setAttribute("role", "button");
      part.tabIndex = 0;
    } else {
      const formatted = display(token);
      const glyph = calculatorOperatorGlyph(token);
      if (glyph !== token) part.dataset.calculationOperator = token;
      const contents = isCalculatorMathGlyph(formatted) ? document.createElement("span") : part;
      if (contents !== part) {
        contents.className = "calculator-math-glyph";
        contents.dataset.calculatorGlyph = formatted;
        part.append(contents);
      }
      for (const piece of formatted.split(/([,万亿兆])/)) {
        if (/^[,万亿兆]$/.test(piece)) {
          // A visual grouping mark occupies space without entering the expression or its caret offsets.
          const mark = document.createElement("span");
          mark.dataset.calculationGrouping = piece;
          mark.contentEditable = "false";
          mark.setAttribute("aria-hidden", "true");
          contents.append(mark);
        } else if (piece) contents.append(piece);
      }
    }
    nodes.push(part);
    start = index + token.length;
  }
  if (start < value.length) nodes.push(document.createTextNode(value.slice(start)));
  field.replaceChildren(...nodes);
  updateLinks(field, operand);
}

function updateLinks(field: HTMLElement, operand: (token: string) => LinkedOperand) {
  for (const chip of field.querySelectorAll<HTMLElement>("[data-calculation-link]")) {
    const { text, title } = operand(chip.dataset.calculationLink!);
    if (chip.textContent !== text) chip.textContent = text;
    if (chip.title !== title) chip.title = title;
    const label = `${text} · ${title}`;
    if (chip.getAttribute("aria-label") !== label) chip.setAttribute("aria-label", label);
  }
}

export function CalculatorExpressionEditor({
  ref, value, operand, display, label, placeholder, busy, onChange, onReject, onKeyDown,
}: {
  ref: Ref<CalculatorEditor>;
  value: string;
  operand: (token: string) => LinkedOperand;
  display: (token: string) => string;
  label: string;
  placeholder: string;
  busy: boolean;
  onChange: (value: string) => void;
  onReject: () => void;
  onKeyDown: (event: KeyboardEvent<HTMLDivElement>) => void;
}) {
  const field = useRef<HTMLDivElement>(null);
  const composing = useRef(false);
  const painted = useRef<{ value: string; display: typeof display } | undefined>(undefined);
  const activeToken = useRef<number | null>(null);
  const lastRange = useRef<Range | null>(null);

  function markToken(node: HTMLElement, offset: number | null) {
    const previous = node.querySelector<HTMLElement>("[data-calculation-selected]");
    const token = offset === null ? null : node.querySelector<HTMLElement>(`[data-calculation-token="${offset}"]`);
    activeToken.current = token ? offset : null;
    if (previous === token) return;
    previous?.removeAttribute("data-calculation-selected");
    previous?.removeAttribute("data-calculation-caret-edge");
    if (previous?.dataset.calculationLink) previous.contentEditable = "false";
    token?.setAttribute("data-calculation-selected", "");
    if (token?.dataset.calculationLink) token.removeAttribute("contenteditable");
  }

  function restoreCaret(node: HTMLElement, next: string, offset: number) {
    if (activeToken.current !== null) {
      const candidates = calculatorExpressionItems(next).filter(item => item.offset <= offset && offset <= item.offset + item.value.length);
      const item = candidates.find(item => item.offset === activeToken.current) ?? candidates.at(-1);
      if (item) {
        const within = offset - item.offset;
        focusToken(item.offset, { offset: item.kind === "reference" && within > 0 ? Infinity : within });
        return;
      }
      markToken(node, null);
    }
    placeCaret(...textPosition(node, offset));
    syncSelection(node);
  }

  function syncSelection(node: HTMLElement) {
    if (document.activeElement !== node || composing.current) return;
    const selection = document.getSelection();
    if (!selection?.rangeCount) return;
    const range = selection.getRangeAt(0);
    if (!node.contains(range.commonAncestorContainer)) return;
    if (activeToken.current !== null && selection.isCollapsed) {
      const anchor = selection.anchorNode;
      const item = calculatorItemAt(anchor instanceof Element ? anchor : anchor?.parentElement ?? null);
      markToken(node, item?.dataset.calculationToken === undefined ? null : Number(item.dataset.calculationToken));
    }
    syncCaretEdge(node);
    lastRange.current = range.cloneRange();
  }

  function focus(options?: FocusOptions) {
    const node = field.current!;
    if (document.activeElement !== node) {
      const range = lastRange.current;
      node.focus(options);
      if (range && node.contains(range.commonAncestorContainer)) selectRange(range);
    }
    syncSelection(node);
  }

  function renderValue(node: HTMLElement, next: string) {
    lastRange.current = null;
    paint(node, next, operand, display);
    painted.current = { value: next, display };
  }

  function focusToken(offset: number, position: CalculatorCaretPosition) {
    const node = field.current!;
    const token = node.querySelector<HTMLElement>(`[data-calculation-token="${offset}"]`);
    if (!token) return;
    node.focus({ preventScroll: true });
    markToken(node, offset);
    if (token.dataset.calculationLink && token.firstChild) {
      const text = token.textContent ?? "";
      let index = 0;
      let digits = 0;
      while (index < text.length && (digits < position.offset || (position.preferNext && groupingMarks.includes(text.charAt(index))))) {
        if (!groupingMarks.includes(text.charAt(index))) digits++;
        index++;
      }
      placeCaret(token.firstChild, index);
    } else {
      placeCaret(...textPosition(token, Math.min(position.offset, raw(token).length), position.preferNext));
    }
    syncSelection(node);
  }

  useLayoutEffect(() => {
    const node = field.current!;
    if (composing.current) return;
    if (painted.current?.value !== value || painted.current.display !== display) {
      const focused = document.activeElement === node;
      const reference = focused && painted.current?.value === value ? referenceSelection(node) : null;
      const end = focused ? selectionOffsets(node)[1] : 0;
      renderValue(node, value);
      if (reference) focusToken(reference.offset, { offset: reference.to });
      else if (focused) restoreCaret(node, value, Math.min(value.length, end));
    } else {
      const reference = document.activeElement === node ? referenceSelection(node) : null;
      const refresh = reference && reference.chip.textContent !== operand(reference.token).text;
      updateLinks(node, operand);
      if (refresh) focusToken(reference.offset, { offset: reference.to });
    }
  }, [value, operand, display]);

  function applyExpression(node: HTMLElement, value: string, caret: number) {
    renderValue(node, value);
    node.focus({ preventScroll: true });
    restoreCaret(node, value, caret);
    onChange(value);
  }

  function replaceRange(node: HTMLElement, before: string, text: string, after: string, caret: number, detach = false) {
    let next = before + text + after;
    if (detach && text.startsWith("-") && !isCalculatorDraft(next)) {
      next = before + "(" + text + ")" + after;
      caret++;
    }
    if (text === "(" && !isCalculatorDraft(next)) {
      next = before + "×(" + after;
      caret++;
    }
    if (!isCalculatorDraft(next)) return false;
    applyExpression(node, next, caret);
    return true;
  }

  function insert(text: string, backspace = false) {
    const node = field.current!;
    const current = raw(node);
    let [start, end] = selectionOffsets(node);
    text = normalizeCalculatorInput(text);
    const numeric = numericInput.test(text);
    const reference = referenceSelection(node);
    if (reference && (numeric || (backspace && reference.to > 0))) {
      const literal = operand(reference.token).value ?? "";
      const to = numeric && reference.from === 0 && reference.to === 0 && literal.startsWith("-") ? 1 : reference.to;
      const from = to !== reference.to ? to : backspace && reference.from === reference.to ? Math.max(0, reference.from - 1) : reference.from;
      const next = literal.slice(0, from) + text + literal.slice(to);
      if (next === literal) return true;
      return replaceRange(node, current.slice(0, reference.offset), next === "-" ? "" : next,
        current.slice(reference.offset + reference.token.length), reference.offset + (next === "-" ? 0 : Math.min(literal.length, from) + text.length), true);
    }
    if (reference) start = end = reference.offset + (reference.to === 0 ? 0 : reference.token.length);
    const linked = !reference && start === end && (backspace || numeric) ? current.slice(0, start).match(referenceAtEnd)?.[0] : undefined;
    if (linked) {
      const literal = operand(linked).value ?? "";
      text = backspace ? literal.slice(0, -1) : literal + text;
      if (text === "-") text = "";
      start -= linked.length;
    } else if (backspace && start === end) {
      if (start === 0) return true;
      start--;
    }
    // Keep an inserted reference separate from any following digits.
    const before = current.slice(0, start);
    const after = current.slice(end);
    if (referenceAtEnd.test(text) && /^[0-9]/.test(after)) text += " ";
    if (/^[0-9]/.test(text || after) && referenceAtEnd.test(before)) text = " " + text;
    return replaceRange(node, before, text, after, start + text.length, !!linked);
  }

  function acceptInput(node: HTMLDivElement, data: string | null) {
    const reference = referenceSelection(node);
    const source = reference && operand(reference.token);
    if (reference && source && reference.chip.textContent !== source.text) {
      const changed = reference.chip.textContent ?? "";
      const inserted = normalizeCalculatorInput(data ?? "");
      const before = value.slice(0, reference.offset);
      const after = value.slice(reference.offset + reference.token.length);
      // Native text insertion can add an operator inside the chip at its visual edge.
      if (/^[+−×÷*/()-]/.test(inserted) && !inserted.includes("#")) {
        const normalized = normalizeCalculatorInput(changed);
        if (normalized === source.text + inserted &&
            replaceRange(node, before + reference.token, inserted, after, reference.offset + reference.token.length + inserted.length)) return;
        if (normalized === inserted + source.text &&
            replaceRange(node, before, inserted, reference.token + after, reference.offset + inserted.length)) return;
      }
      const text = normalizeCalculatorInput(withoutGrouping(changed));
      const literal = text === "-" ? "" : text;
      const numericEdit = (!data || numericInput.test(inserted)) && /^-?[0-9.]*$/.test(literal);
      if (numericEdit && literal !== source.value && replaceRange(node, before, literal, after,
        reference.offset + Math.min(literal.length, reference.to), true)) return;
      renderValue(node, value);
      focusToken(reference.offset, { offset: reference.from });
      if (!numericEdit || literal !== source.value) onReject();
      return;
    }
    const next = normalizeCalculatorInput(raw(node));
    const end = selectionOffsets(node)[1];
    const typedReference = hasTypedReference(node);
    if (!typedReference && isCalculatorDraft(next)) {
      renderValue(node, next);
      restoreCaret(node, next, Math.min(end, next.length));
      if (next !== value) onChange(next);
      return;
    }
    const offset = end - Math.max(0, next.length - value.length);
    // Composition and native text insertion can deliver input after the non-editable reference.
    const inserted = normalizeCalculatorInput(data ?? "");
    const fragment = next.slice(offset, end);
    const insertedAtCaret = !typedReference && next.slice(0, offset) + next.slice(end) === value &&
      (fragment === inserted || fragment === " " + inserted);
    const appended = insertedAtCaret && (inserted === "(" || (numericInput.test(inserted) && referenceAtEnd.test(value.slice(0, offset))));
    renderValue(node, value);
    restoreCaret(node, value, Math.max(0, Math.min(offset, value.length)));
    if (appended && insert(inserted)) return;
    onReject();
  }

  useImperativeHandle(ref, () => ({
    focus,
    focusEnd: () => {
      const node = field.current!;
      node.focus({ preventScroll: true });
      placeCaret(...textPosition(node, raw(node).length));
      syncSelection(node);
    },
    focusToken,
    scrollIntoView: options => field.current?.scrollIntoView(options),
    insert,
    replayBackspaces: count => {
      const node = field.current!;
      while (count > 0 && raw(node)) {
        if (!insert("", true)) { onReject(); return 0; }
        count--;
      }
      return count;
    },
    removeItem: () => {
      const node = field.current!;
      if (activeToken.current === null) return false;
      const next = calculatorItemDeletion(raw(node), activeToken.current);
      if (!next) return false;
      applyExpression(node, next.expression, next.caret);
      return true;
    },
  }));

  return <div id="calculator-expression" className="calculator-expression-input" ref={field} role="textbox"
    contentEditable={!busy} tabIndex={0} aria-label={label} aria-placeholder={placeholder} aria-readonly={busy} aria-multiline={false}
    data-placeholder={placeholder} data-empty={!value} spellCheck={false}
    onSelect={event => syncSelection(event.currentTarget)}
    onInput={event => { if (!busy && !composing.current) acceptInput(event.currentTarget, (event.nativeEvent as InputEvent).data); }}
    onCompositionStart={() => { composing.current = true; }}
    onCompositionEnd={event => { composing.current = false; if (!busy) acceptInput(event.currentTarget, event.data); }}
    onKeyDown={event => {
      const node = event.currentTarget;
      const key = calculatorInputKey(event.key, event.code);
      const plainKey = !busy && !event.nativeEvent.isComposing && !event.metaKey && !event.ctrlKey && !event.altKey;
      const reference = plainKey ? referenceSelection(node) : null;
      if (plainKey && numericInput.test(key)) {
        const [start, end] = selectionOffsets(node);
        if (key === "." || reference || (start === end && referenceAtEnd.test(raw(node).slice(0, start)))) {
          event.preventDefault();
          if (!insert(key)) onReject();
          return;
        }
      }
      onKeyDown(event);
      if (!event.defaultPrevented && plainKey && (/^[()]$/.test(key) || (reference && /^[+−×÷*/-]$/.test(key)))) {
        event.preventDefault();
        if (!insert(key)) onReject();
      }
    }}
    onClick={event => {
      if (event.detail > 1 || event.shiftKey) return;
      if (!calculatorItemAt(event.target)) { markToken(event.currentTarget, null); syncSelection(event.currentTarget); }
    }}
    onPaste={event => {
      event.preventDefault();
      if (busy) return;
      const text = event.clipboardData.getData("text/plain");
      if (text.includes("#") || !insert(text)) onReject();
    }}
    onDrop={event => event.preventDefault()} />;
}
