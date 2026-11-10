import { memo, useCallback, useEffect, useEffectEvent, useLayoutEffect, useMemo, useRef, useState, type MouseEvent } from "react";
import { useTranslation } from "react-i18next";
import { Channel, isMyGo } from "mygo-runtime";
import { System, events, type CalculationEntry, type CalculationValue, type CalculatorFormat, type CalculatorSheet } from "./mygo";
import { icon } from "./icons";
import { calculatorOperatorGlyph, formatCalculation, formatCalculatorToken, isCalculatorMathGlyph } from "./calculation-format";
import { CalculationText } from "./calculation-text";
import { CalculatorExpressionEditor, calculatorTokenPosition, type CalculatorCaretPosition, type CalculatorEditor, type LinkedOperand } from "./calculator-expression-editor";
import { useCalculatorConnections, type CalculatorTrace } from "./calculator-connection";
import { CalculatorSettings } from "./calculator-settings";
import { calculatorInputKey, isCalculatorDraft, normalizeCalculatorInput } from "./calculator-input";
import { calculatorExpressionItems, calculatorItemAt, calculatorReferenceBindings, calculatorResultItems, remapCalculatorReferences } from "./calculator-items";

const keys = ["AC", "(", ")", "÷", "7", "8", "9", "×", "4", "5", "6", "−", "1", "2", "3", "+", "0", ".", "Backspace", "="];
const keyGlyphs: Partial<Record<string, string>> = {
  "(": "M13.25 4.5a12.5 12.5 0 0 0 0 15",
  ")": "M10.75 4.5a12.5 12.5 0 0 1 0 15",
  "÷": "M5.5 12h13 M12 6h.01 M12 18h.01",
  "×": "m7 7 10 10 M17 7 7 17",
  "−": "M5.5 12h13",
  "+": "M5.5 12h13 M12 5.5v13",
  "=": "M5.5 9h13 M5.5 15h13",
};
const Icon = ({ name }: { name: string }) => <span className="inline-flex" aria-hidden="true" dangerouslySetInnerHTML={{ __html: icon(name) }} />;
const compactExpression = (value: string) => value.replace(/\s/g, "");
type ItemFocus = { offset: number; position: CalculatorCaretPosition };
type ResultItem = "answer" | "equality";
type ResultSelection = { id: string; item: ResultItem; active: boolean; resumeAfterDelete?: boolean };
export type CalculatorImport = { expression: string };

const calculationErrors: Record<string, string> = {
  invalid_expression: "请输入完整的算式。",
  division_by_zero: "不能除以零。",
  expression_too_long: "算式太长，请缩短后再试。",
  calculation_not_found: "这条计算记录已不存在。",
  calculation_history_changed: "计算历史已变更，请检查引用后重试。",
  invalid_calculation_reference: "只能引用此行上方的结果。",
  invalid_calculation_dependency: "引用的结果不可用。",
  result_too_large: "结果太大，请缩短计算链。",
};

const Expression = memo(function Expression({ value, display, operand, editable }: { value: string; display: (token: string) => string; operand?: (token: string) => LinkedOperand; editable?: boolean }) {
  return calculatorExpressionItems(value).map(({ value: token, offset, kind }) => {
    const linkedValue = kind === "reference" ? operand?.(token) : undefined;
    const text = linkedValue?.text ?? display(token);
    const content = isCalculatorMathGlyph(text) ? <span className="calculator-math-glyph" data-calculator-glyph={text}>{text}</span> : <CalculationText text={text} />;
    return editable ? <button key={offset} type="button" className={linkedValue ? "calculator-reference" : undefined}
      data-calculation-item={kind} data-calculation-token={offset} data-calculation-link={linkedValue ? token : undefined}
      title={linkedValue?.title} aria-label={linkedValue ? `${linkedValue.text} · ${linkedValue.title}` : undefined}>{content}</button> : <span key={offset} className={linkedValue ? "calculator-reference" : undefined}
      data-calculation-item={kind} data-calculation-token={offset} data-calculation-link={linkedValue ? token : undefined}>{content}</span>;
  });
});

const ResultItems = memo(function ResultItems({ value, approximate, grouping, source, selected }: {
  value: string;
  approximate: boolean;
  grouping?: string;
  source?: number;
  selected?: ResultItem;
}) {
  const { t, i18n } = useTranslation();
  const result = { value, approximate };
  const [equality, answer] = calculatorResultItems(result);
  const text = formatCalculation({ value: answer.value, approximate: false }, i18n.language, grouping);
  const content = <CalculationText text={text} />;
  if (source === undefined) return <><span data-calculation-item={equality.kind}>{equality.value}</span>
    <strong className="calculator-use-result" data-calculation-selected={selected === "answer" ? "" : undefined}>{content}</strong></>;
  const title = t("选中此结果，按运算符引用");
  return <>
    <button type="button" data-calculation-item={equality.kind} title={title}
      aria-pressed={selected === equality.kind} aria-label={t("选中第 {{row}} 行的 {{sign}}", { sign: equality.value, row: source })}>{equality.value}</button>
    <button type="button" className="calculator-use-result" data-calculation-item={answer.kind}
      aria-label={t("选中第 {{row}} 行结果 {{result}}", { row: source, result: formatCalculation(result, i18n.language, grouping) })}
      aria-pressed={selected === answer.kind} title={title}>
      <strong data-calculation-source={source}>{content}</strong>
    </button>
  </>;
});

export function CalculatorView({ initialExpression, initialRowIDs, incoming, onImportHandled, onEdit }: {
  initialExpression: string;
  initialRowIDs?: readonly string[];
  incoming?: CalculatorImport;
  onImportHandled: (request: CalculatorImport) => void;
  onEdit: (expression: string, rowIDs: readonly string[]) => void;
}) {
  const { t, i18n } = useTranslation();
  const native = isMyGo();
  const input = useRef<CalculatorEditor>(null);
  const canvas = useRef<HTMLDivElement>(null);
  const focusToken = useRef<ItemFocus | null>(null);
  const focusAnswer = useRef<string | null>(null);
  const focusImportedDraft = useRef(false);
  const returnToRow = useRef<string | undefined>(undefined);
  const revealEditor = useRef(false);
  const savedScrollTop = useRef<number | undefined>(undefined);
  const followHistory = useRef(true);
  const historyRequest = useRef(0);
  const pendingDeletion = useRef<{ repeats: number } | null>(null);
  const [selectedResult, setSelectedResult] = useState<ResultSelection | null>(null);
  const [draft, setDraft] = useState(() => compactExpression(initialExpression));
  const [savingAnswer, setSavingAnswer] = useState<CalculationValue>();
  const [draftOpen, setDraftOpen] = useState(true);
  const [editing, setEditing] = useState<{ id: string; expression: string }>();
  const [preview, setPreview] = useState<{ id: string; expression: string; sheet: CalculatorSheet | undefined; rows: CalculationEntry[] | null }>();
  const [sheet, setSheet] = useState<CalculatorSheet>();
  const [historyError, setHistoryError] = useState(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ key: string; error?: boolean }>();
  const [trace, setTrace] = useState<CalculatorTrace | null>(null);
  const history = sheet?.history;
  const grouping = sheet?.format.grouping;
  const displayToken = useCallback((token: string) => formatCalculatorToken(token, i18n.language, grouping), [i18n.language, grouping]);
  const expression = editing?.expression ?? draft;
  const savedRows = useMemo(() => [...(history ?? [])].reverse(), [history]);
  const rowIDs = useMemo(() => savedRows.map(row => row.createdAt), [savedRows]);
  const latest = useRef({ draft, editing, rows: initialRowIDs ?? rowIDs, onEdit });
  useLayoutEffect(() => {
    latest.current = { draft, editing, rows: history ? rowIDs : initialRowIDs ?? rowIDs, onEdit };
  }, [draft, editing, history, initialRowIDs, rowIDs, onEdit]);
  const editingIndex = savedRows.findIndex(row => row.createdAt === editing?.id);
  const editingEntry = savedRows[editingIndex];
  const editingChanged = !!editing && expression !== compactExpression(editingEntry?.expression ?? "");
  const activePreview = editingChanged && preview?.id === editing?.id && preview?.expression === expression && preview.sheet === sheet ? preview : undefined;
  const historyRows = useMemo(() => {
    if (!activePreview?.rows) return savedRows;
    const liveRows = new Map(activePreview.rows.map(row => [row.createdAt, row]));
    return savedRows.map(row => liveRows.get(row.createdAt) ?? row);
  }, [savedRows, activePreview]);
  const editedRow = editingChanged ? activePreview?.rows?.at(-1) : editingEntry;
  const editedResult = editedRow?.error || editedRow?.pending ? undefined : editedRow?.result;
  const displayedResult = savingAnswer ?? editedResult;
  const savedError = !editingChanged ? editingEntry?.error : undefined;
  const connection = useCalculatorConnections(canvas, trace);
  const refreshHistory = useCallback(async () => {
    const request = ++historyRequest.current;
    try {
      const next = await System.calculatorSheet();
      if (request !== historyRequest.current) return;
      const previous = latest.current;
      const rows = next.history.map(row => row.createdAt).reverse();
      if (previous.editing && !rows.includes(previous.editing.id)) {
        setHistoryError(true);
        setMessage({ key: calculationErrors.calculation_not_found!, error: true });
        return;
      }
      const draft = remapCalculatorReferences(previous.draft, previous.rows, rows);
      const editing = previous.editing && { ...previous.editing, expression: remapCalculatorReferences(previous.editing.expression, previous.rows, rows) };
      latest.current = { ...previous, draft, editing, rows };
      setDraft(draft);
      previous.onEdit(draft, rows);
      if (editing?.expression !== previous.editing?.expression) setEditing(editing);
      setSheet(next);
      setHistoryError(false);
    } catch (reason) {
      if (request === historyRequest.current) { console.error(reason); setHistoryError(true); }
    }
  }, []);

  const linkedOperand = useCallback((token: string, rows = historyRows): LinkedOperand => {
    const row = Number(token.slice(1));
    const source = rows[row - 1];
    if (!source || source.error || source.pending) return { text: "—", title: t("引用的结果不可用。") };
    const text = formatCalculation({ ...source.result, approximate: false }, i18n.language, grouping);
    return { text, value: source.result.value, title: t("来自第 {{row}} 行", { row }) };
  }, [historyRows, i18n.language, grouping, t]);

  useLayoutEffect(() => {
    const row = returnToRow.current;
    const answer = focusAnswer.current && canvas.current?.querySelector<HTMLElement>(
      `[data-calculation-row="${CSS.escape(focusAnswer.current)}"] [data-calculation-item="answer"]`,
    );
    if (answer) answer.focus({ preventScroll: true });
    else if (pendingDeletion.current && !editing) {
      canvas.current?.focus({ preventScroll: true });
    } else if (!editing && row && !revealEditor.current) {
      canvas.current?.querySelector<HTMLElement>(`[data-calculation-row="${CSS.escape(row)}"] [data-calculation-token]`)?.focus({ preventScroll: true });
    } else if (focusToken.current) {
      const { offset, position } = focusToken.current;
      input.current?.focusToken(offset, position);
      focusToken.current = null;
    } else if (editing || draftOpen) input.current?.focusEnd();
    else canvas.current?.focus({ preventScroll: true });
    if (canvas.current && savedScrollTop.current !== undefined) canvas.current.scrollTop = savedScrollTop.current;
    if (answer) answer.scrollIntoView({ block: "nearest" });
    if (revealEditor.current) input.current?.scrollIntoView({ block: "nearest" });
    focusAnswer.current = null;
    revealEditor.current = false;
    returnToRow.current = undefined;
    savedScrollTop.current = undefined;
  }, [editing?.id, draftOpen]);
  useLayoutEffect(() => {
    if (!focusImportedDraft.current) return;
    focusImportedDraft.current = false;
    input.current?.focusEnd();
    input.current?.scrollIntoView({ block: "nearest" });
  }, [draft]);
  useLayoutEffect(() => {
    if (busy || !pendingDeletion.current) return;
    const { repeats } = pendingDeletion.current;
    pendingDeletion.current = null;
    if (!repeats) return;
    if (selectedResult?.active) { removeSelectedResult(repeats - 1); return; }
    if (!editing) return;
    const remaining = input.current?.replayBackspaces(repeats) ?? 0;
    // A repeat on the now-empty row removes it before the next handoff.
    if (remaining > 0) deleteEmptyRow(remaining - 1);
  }, [busy, editing?.id, selectedResult]);
  useLayoutEffect(() => {
    if (history && followHistory.current && canvas.current) {
      canvas.current.scrollTop = canvas.current.scrollHeight;
      followHistory.current = false;
    }
  }, [history]);
  useEffect(() => {
    if (!native) return;
    const unsubscribe = events.calculatorHistoryChanged.on(() => {
      const view = canvas.current;
      followHistory.current = !!view && view.scrollHeight - view.scrollTop - view.clientHeight < 40;
      void refreshHistory();
    });
    void refreshHistory();
    return () => { unsubscribe(); historyRequest.current++; };
  }, [native, refreshHistory]);
  useEffect(() => {
    const id = editing?.id;
    if (!native || !id || !editingChanged || editingEntry?.pending) return;
    let active = true;
    void System.calculatorPreviewEdit(id, expression, calculatorReferenceBindings(expression, rowIDs)).then(rows => {
      if (active) setPreview({ id, expression, sheet, rows });
    }).catch(reason => {
      if (active) { console.error(reason); setPreview({ id, expression, sheet, rows: null }); }
    });
    return () => { active = false; };
  }, [expression, native, editing?.id, editingChanged, editingEntry?.pending, sheet, rowIDs]);
  const applyImport = useEffectEvent((request: CalculatorImport) => {
    if (editing) changeEditing();
    setTrace(null);
    focusImportedDraft.current = true;
    const value = compactExpression(request.expression);
    setSelectedResult(null);
    setDraftOpen(true);
    updateDraft(value);
    onImportHandled(request);
  });
  useEffect(() => {
    if (incoming && !busy && !draft && !editingChanged) applyImport(incoming);
  }, [incoming, busy, draft, editingChanged]);
  function updateDraft(value: string, rows = history ? rowIDs : initialRowIDs ?? rowIDs) {
    latest.current = { ...latest.current, draft: value, rows };
    setDraft(value);
    onEdit(value, rows);
  }

  function edit(value: string) {
    setSelectedResult(null);
    if (trace?.target === (editing?.id ?? "draft")) {
      const references = (value.match(/#[1-9][0-9]*/g) ?? []).filter(token => token === `#${trace.source}`);
      if (references.length <= trace.occurrence) setTrace(null);
    }
    if (editing) setEditing({ ...editing, expression: value });
    else {
      if (!draftOpen) revealEditor.current = true;
      setDraftOpen(true);
      updateDraft(value);
    }
    setMessage(undefined);
  }

  function changeEditing(entry?: CalculationEntry, offset?: number, position: CalculatorCaretPosition = { offset: 0 }) {
    setSelectedResult(current => {
      if (!entry || !current) return null;
      const source = savedRows.findIndex(row => row.createdAt === current.id && !row.error && !row.pending);
      const target = savedRows.findIndex(row => row.createdAt === entry.createdAt);
      return source >= 0 && source < target ? { ...current, active: false } : null;
    });
    savedScrollTop.current = canvas.current?.scrollTop;
    const value = entry ? compactExpression(entry.expression) : "";
    focusToken.current = offset === undefined ? null : { offset, position };
    returnToRow.current = entry ? undefined : editing?.id;
    setEditing(entry ? { id: entry.createdAt, expression: value } : undefined);
    setPreview(undefined);
    setMessage(undefined);
  }

  async function perform(action: () => Promise<void>, failure: string, pending?: string) {
    if (busy) return;
    setBusy(true);
    setMessage(pending ? { key: pending } : undefined);
    try {
      await action();
    } catch (reason) {
      const code = reason instanceof Error ? reason.message : String(reason);
      const key = calculationErrors[code];
      if (!key) console.error(reason);
      if (code === "calculation_history_changed") await refreshHistory();
      setMessage({ key: key ?? failure, error: true });
    } finally {
      setBusy(false);
    }
  }

  function applySheet(updatedSheet: CalculatorSheet, request: number) {
    const concurrentRefresh = request !== historyRequest.current;
    historyRequest.current++;
    const previous = latest.current;
    const rows = updatedSheet.history.map(row => row.createdAt).reverse();
    updateDraft(remapCalculatorReferences(previous.draft, previous.rows, rows), rows);
    if (previous.editing && rows.includes(previous.editing.id)) {
      const expression = remapCalculatorReferences(previous.editing.expression, previous.rows, rows);
      if (expression !== previous.editing.expression) {
        const editing = { ...previous.editing, expression };
        latest.current.editing = editing;
        setEditing(editing);
      }
    }
    setSheet(updatedSheet);
    setHistoryError(false);
    if (concurrentRefresh) void refreshHistory();
  }

  function commit(afterEdit?: () => void) {
    if (!native || busy) return;
    const finishCommit = (entry?: CalculationEntry) => {
      if (afterEdit) { afterEdit(); return; }
      if (editing) changeEditing();
      if (!editing || !draft) setDraftOpen(false);
      returnToRow.current = undefined;
      if (entry && !entry.error) {
        focusAnswer.current = entry.createdAt;
        setSelectedResult({ id: entry.createdAt, item: "answer", active: true });
      }
    };
    if (editing && !editingChanged && !editingEntry?.pending) { finishCommit(editingEntry); return; }
    if (afterEdit && editingEntry?.pending) { setMessage({ key: "按回车计算，或按 Esc 取消编辑。" }); return; }
    if (!expression) {
      if (editing) setMessage({ key: "请输入完整的算式。", error: true });
      return;
    }
    void perform(async () => {
      if (!afterEdit && editedResult) setSavingAnswer(editedResult);
      const request = historyRequest.current;
      const binding = calculatorReferenceBindings(expression, history ? rowIDs : initialRowIDs ?? rowIDs);
      const { entry, sheet: updatedSheet } = editing && afterEdit ? await System.calculatorSave(expression, editing.id, binding) :
        await System.calculatorSaveWithAnswer(expression, editing?.id ?? null, new Channel<CalculationValue>(setSavingAnswer), binding).catch(reason => {
          setSavingAnswer(undefined);
          throw reason;
        });
      if (!editing) followHistory.current = true;
      applySheet(updatedSheet, request);
      setSavingAnswer(undefined);
      if (!editing) {
        setTrace(current => current?.target === "draft" ? { ...current, target: entry.createdAt } : current);
        updateDraft("");
      }
      finishCommit(entry);
      setMessage({ key: editing ? "已更新原记录" : "已存入计算历史" });
    }, "无法保存计算，请重试。", afterEdit ? undefined : "正在计算…");
  }

  function selectHistoryToken(entry: CalculationEntry, offset: number, position: CalculatorCaretPosition) {
    const select = () => changeEditing(entry, offset, position);
    if (editingChanged) commit(select);
    else select();
  }

  function startDraft(item?: ItemFocus) {
    if (busy) return;
    const show = () => {
      setSelectedResult(null);
      setTrace(null);
      setDraftOpen(true);
      if (editing || !draftOpen) {
        revealEditor.current = true;
        if (editing) changeEditing();
        focusToken.current = item ?? null;
      } else {
        input.current?.focus({ preventScroll: true });
        input.current?.scrollIntoView({ block: "nearest" });
      }
    };
    if (editingChanged) commit(show);
    else show();
  }

  function copy(value: string) {
    void perform(async () => {
      await System.calculatorCopy(value);
      setMessage({ key: "已复制计算结果" });
    }, "无法复制计算结果，请重试。");
  }

  async function clear() {
    await System.calculatorClearHistory();
    setSelectedResult(null);
    setTrace(null);
    setEditing(undefined);
    setDraftOpen(true);
    historyRequest.current++;
    latest.current = { ...latest.current, rows: [], editing: undefined };
    if (draft.includes("#")) {
      updateDraft("", []);
    }
    setSheet(current => current && { ...current, history: [] });
    void refreshHistory();
  }

  async function saveFormat(format: CalculatorFormat) {
    await System.saveCalculatorFormat(format);
    setSheet(current => current && { ...current, format });
    void refreshHistory();
  }

  function traceChip(chip: HTMLElement) {
    const token = chip.dataset.calculationLink!;
    if (token === "#0") return;
    const target = chip.closest<HTMLElement>("[data-calculation-row]");
    if (!target) return;
    const occurrence = [...target.querySelectorAll(`[data-calculation-link="${CSS.escape(token)}"]`)].indexOf(chip);
    const next = { source: Number(token.slice(1)), target: target.dataset.calculationRow!, occurrence };
    setTrace(current => current?.source === next.source && current.target === next.target && current.occurrence === occurrence ? null : next);
  }

  function selectResult(id: string, rowIndex: number, item: ResultItem, keepSelected = false) {
    const selection = { id, item, active: true };
    if (editing && rowIndex >= editingIndex) {
      // Continue in the draft when this result would be a self or forward reference in the edit.
      const select = () => {
        changeEditing();
        returnToRow.current = undefined;
        setSelectedResult(selection);
      };
      if (editingChanged) commit(select);
      else select();
      return;
    }
    setSelectedResult(current => current?.id === id && current.item === item && current.active
      ? (keepSelected ? current : null) : selection);
    setMessage(undefined);
    (input.current ?? canvas.current)?.focus({ preventScroll: true });
  }

  function activateItem(event: MouseEvent<HTMLDivElement>) {
    if (busy || !(event.target instanceof Element)) return;
    const target = event.target;
    const item = calculatorItemAt(target);
    if (!item && target.closest(".calculator-draft-entry")) { startDraft(); return; }
    const row = target.closest<HTMLElement>("[data-calculation-row]");
    const rowIndex = row ? savedRows.findIndex(entry => entry.createdAt === row.dataset.calculationRow) : -1;
    const entry = savedRows[rowIndex];
    if (!item) {
      if (!entry || target.closest("button, .calculator-expression-input")) return;
      const answer = row?.querySelector<HTMLButtonElement>('button[data-calculation-item="answer"]');
      if (answer && event.clientX >= answer.getBoundingClientRect().right) {
        event.stopPropagation();
        selectResult(entry.createdAt, rowIndex, "answer", true);
      }
      return;
    }
    const kind = item.dataset.calculationItem;
    const inEditor = item.closest("#calculator-expression");
    if (inEditor && (event.detail > 1 || event.shiftKey)) return;
    if (kind === "answer" || kind === "equality") {
      if (entry) selectResult(entry.createdAt, rowIndex, kind);
      return;
    }
    setSelectedResult(current => current?.active ? { ...current, active: false } : current);
    const offset = Number(item.dataset.calculationToken);
    const position = event.detail ? calculatorTokenPosition(item, event.clientX, event.clientY) : { offset: Infinity };
    if (item.closest(".calculator-draft-entry")) { startDraft({ offset, position }); return; }
    if (kind === "reference") traceChip(item);
    if (inEditor) input.current?.focusToken(offset, position);
    else if (entry) selectHistoryToken(entry, offset, position);
  }

  function continueResult(key: string) {
    const operator = calculatorOperatorGlyph(normalizeCalculatorInput(key));
    if (busy || !/^[+−×÷()]$/.test(operator)) return false;
    const id = selectedResult?.id;
    const source = id ? historyRows.findIndex(row => row.createdAt === id && !row.error && !row.pending) + 1 : 0;
    if (!source) return false;
    const operand = `#${source}`;
    const join = operator === "(" ? "×(" : operator;
    // The caret can follow a complete operand or already be waiting for one.
    const inserted = insert(operand + join) || (operator !== ")" && insert(join + operand));
    if (inserted) input.current?.scrollIntoView({ block: "nearest" });
    else rejectInput();
    return true;
  }

  function insert(text: string, backspace = false) {
    if (input.current) return input.current.insert(text, backspace);
    if (backspace) return true;
    const value = normalizeCalculatorInput(text);
    if (!isCalculatorDraft(value)) return false;
    edit(value);
    return true;
  }

  function performDeletion(action: () => Promise<void>, failure: string, repeats: number) {
    pendingDeletion.current = { repeats };
    void perform(async () => {
      try { await action(); }
      catch (reason) { pendingDeletion.current = null; throw reason; }
    }, failure);
  }

  function removeSelectedResult(repeats = 0) {
    if (!selectedResult?.active) return false;
    if (busy || pendingDeletion.current) return true;
    const equality = selectedResult.item === "equality";
    const displayed = historyRows.find(row => row.createdAt === selectedResult.id)?.result.value ?? "";
    const resume = !editing && (equality || !draft || selectedResult.resumeAfterDelete);
    const remove = async () => {
      const request = historyRequest.current;
      const next = await (equality ? System.calculatorDeleteEquality(selectedResult.id, displayed) : System.calculatorDeleteResult(selectedResult.id));
      applySheet(next, request);
      if (resume) changeEditing(next.history.find(row => row.createdAt === selectedResult.id));
      setSelectedResult(null);
      setMessage({ key: equality ? "已删除等号，可按回车重新计算。" : "已移除结果，可按回车重新计算。" });
    };
    const failure = equality ? "无法删除等号，请重试。" : "无法移除计算结果，请重试。";
    if (resume) performDeletion(remove, failure, repeats);
    else void perform(remove, failure);
    return true;
  }

  function deleteEmptyRow(repeats = 0) {
    const previous = editing ? savedRows[editingIndex - 1] : savedRows.at(-1);
    const resume = () => {
      const answer = previous && !previous.pending && !previous.error ? previous : undefined;
      changeEditing(answer ? undefined : previous);
      returnToRow.current = undefined;
      revealEditor.current = !!previous && !answer;
      focusAnswer.current = answer?.createdAt ?? null;
      setSelectedResult(answer ? { id: answer.createdAt, item: "answer", active: true, resumeAfterDelete: true } : null);
      setTrace(null);
    };
    if (!editing) {
      setDraftOpen(false);
      resume();
      return;
    }
    performDeletion(async () => {
      const request = historyRequest.current;
      const next = await System.calculatorDeleteRow(editing.id);
      applySheet(next, request);
      resume();
      setMessage({ key: "已删除空行。" });
    }, "无法删除计算行，请重试。", repeats);
  }

  function removeEmptyRow() {
    if (expression) return false;
    if (!busy && !pendingDeletion.current) deleteEmptyRow();
    return true;
  }

  function removeItem() {
    return removeSelectedResult() || removeEmptyRow() || busy || input.current?.removeItem() === true;
  }

  function press(key: string) {
    if (busy) return;
    if (key === "Backspace" && (removeSelectedResult() || removeEmptyRow())) return;
    if (key === "AC") { setTrace(null); edit(""); input.current?.focus(); }
    else if (!continueResult(key) && !insert(key === "Backspace" ? "" : key, key === "Backspace")) rejectInput();
  }

  function rejectInput() {
    setMessage({ key: "请输入有效的数字、运算符或括号。", error: true });
  }

  const previewError = activePreview?.rows === null;
  const feedback = savingAnswer ? t("正在保存计算记录…") : message ? t(message.key) : previewError ? t("计算预览失败，请重试。") : "";
  const editor = (row: number) => <div className="calculator-line calculator-editor">
    <span className="calculator-row-number" data-calculation-source={editing && !editedResult && !savedError ? row : undefined} aria-hidden="true">{row}</span>
    <CalculatorExpressionEditor ref={input} label={t("算式")} value={expression} operand={linkedOperand} display={displayToken} busy={busy}
      onChange={edit} onReject={rejectInput} onKeyDown={event => {
        if (event.nativeEvent.isComposing) return;
        if (busy || pendingDeletion.current) {
          if (event.key === "Escape" || event.key === "Enter" || event.key === "=" || event.key === "＝" || event.key === "Delete" || event.key === "Backspace") event.preventDefault();
          return;
        }
        if (event.key === "Delete" && !event.altKey && !event.metaKey && !event.ctrlKey && removeItem()) { event.preventDefault(); return; }
        const chip = calculatorItemAt(event.target);
        if (chip?.dataset.calculationItem === "reference" && (event.key === "Enter" || event.key === " ")) {
          event.preventDefault(); traceChip(chip); return;
        }
        if (!event.metaKey && !event.ctrlKey && !event.altKey && continueResult(event.key)) {
          event.preventDefault(); return;
        }
        if (event.key === "Enter" || event.key === "=" || event.key === "＝") { event.preventDefault(); commit(); }
        if (event.key === "Escape") { event.preventDefault(); if (selectedResult) setSelectedResult(null); else if (editing) changeEditing(); else if (trace) setTrace(null); else edit(""); }
        if (event.key === "Backspace" && !busy && !event.altKey && !event.metaKey && !event.ctrlKey) {
          event.preventDefault();
          press("Backspace");
        }
      }} placeholder={t("输入算式…")} />
    {displayedResult ? <output className="calculator-line-result" htmlFor="calculator-expression" aria-label={t("计算结果")}>
      <ResultItems value={displayedResult.value} approximate={displayedResult.approximate} grouping={grouping}
        source={editing && !savingAnswer ? row : undefined} selected={savingAnswer ? "answer" : undefined} />
    </output> : savedError && <span className="calculator-row-error" data-calculation-source={row}>{t(calculationErrors[savedError] ?? "引用的结果不可用。")}</span>}
    {editing && <button type="button" className="icon-button calculator-line-action" disabled={busy} aria-label={t("取消编辑")} title={t("取消编辑")} onClick={() => changeEditing()}><Icon name="close" /></button>}
  </div>;

  return <form className="calculator-page" aria-label={t("计算器")} aria-busy={busy} onSubmit={event => { event.preventDefault(); commit(); }}
    onKeyDownCapture={event => {
      if (!pendingDeletion.current || event.key !== "Backspace" || event.nativeEvent.isComposing || event.metaKey || event.ctrlKey || event.altKey) return;
      event.preventDefault();
      event.stopPropagation();
      if (event.repeat) pendingDeletion.current.repeats++;
    }}
    onClick={event => {
      const target = event.target as HTMLElement;
      if (target === event.currentTarget || (canvas.current?.contains(target) &&
          !target.closest("button, .calculator-line-result, .calculator-expression-input, .calculator-row-error, .calculator-history-note"))) startDraft();
    }}
    onKeyDown={event => { if (event.key === "Escape" && !event.defaultPrevented && trace) { event.preventDefault(); setTrace(null); } }}>
    <CalculatorSettings format={sheet?.format} maxDecimals={sheet?.maxDecimalPlaces ?? 0} hasHistory={!!history?.length} busy={busy || !native}
      onFormat={saveFormat} onClear={clear} actions={connection.hidden && <button type="button" className="calculator-jump" onClick={connection.jump}>
        {connection.hidden === "source" ? t("跳到源结果") : t("返回引用位置")}
      </button>} />
    <div className="calculator-paper">
      <div className="calculator-canvas" ref={canvas} role="region" aria-label={t("计算历史")} tabIndex={-1} onClick={activateItem}
        onKeyDown={event => {
          const item = calculatorItemAt(event.target);
          const onAnswer = selectedResult?.active && item?.dataset.calculationItem === "answer" && item.getAttribute("aria-pressed") === "true";
          if ((!onAnswer && event.target !== event.currentTarget) || editing || event.nativeEvent.isComposing || event.metaKey || event.ctrlKey || event.altKey) return;
          const key = calculatorInputKey(event.key, event.code);
          if (onAnswer && key === "Enter" && event.repeat) { event.preventDefault(); return; }
          if (onAnswer && (key === "Backspace" || key === "Delete") && removeSelectedResult()) { event.preventDefault(); return; }
          if (key === "Backspace" && event.repeat) { event.preventDefault(); return; }
          if (draftOpen && !onAnswer) return;
          if (/^[0-9.+−×÷*/()-]$/.test(key)) { event.preventDefault(); press(key); }
          if ((key === "Delete" || key === "Backspace") && removeSelectedResult()) event.preventDefault();
          if (key === "Escape") { setSelectedResult(null); setTrace(null); if (onAnswer) (input.current ?? canvas.current)?.focus({ preventScroll: true }); }
        }}
        onPaste={event => {
          const item = calculatorItemAt(event.target);
          const onAnswer = item?.dataset.calculationItem === "answer" && item.getAttribute("aria-pressed") === "true";
          if (busy || editing || (!onAnswer && (event.target !== event.currentTarget || draftOpen))) return;
          event.preventDefault();
          const text = event.clipboardData.getData("text/plain");
          if (text.includes("#") || !insert(text)) rejectInput();
        }}
        onMouseDown={event => { if (calculatorItemAt(event.target) instanceof HTMLButtonElement) event.preventDefault(); }}>
        {historyError && <div className="calculator-history-note" role="alert">{t("无法读取计算历史。")} <button type="button" className="secondary-button" onClick={refreshHistory}>{t("重试")}</button></div>}
        {!historyError && !history?.length && <p className="calculator-history-note">{!native ? t("计算历史仅在桌面应用中可用。") : history ? t("按回车计算，或从启动器复制结果，即可留下记录。") : t("正在读取计算历史…")}</p>}
        <div className="calculator-sheet">
          <svg className="calculator-connection" ref={connection.layer} aria-hidden="true" />
          <ol className="calculator-history">
            {historyRows.map((entry, index) => <li key={entry.createdAt} data-calculation-row={entry.createdAt}>
              {editing?.id === entry.createdAt ? editor(index + 1) : <div className="calculator-line">
                <span className="calculator-row-number" data-calculation-source={entry.pending ? index + 1 : undefined} aria-hidden="true">{index + 1}</span>
                <div className="calculator-record">
                  <Expression value={compactExpression(entry.expression)} display={displayToken} operand={entry.expression.includes("#") ? linkedOperand : undefined} editable />
                </div>
                {entry.pending ? null : entry.error ? <span className="calculator-row-error" data-calculation-source={index + 1}>{t(calculationErrors[entry.error] ?? "引用的结果不可用。")}</span> : <div className="calculator-line-result">
                  <ResultItems value={entry.result.value} approximate={entry.result.approximate} grouping={grouping} source={index + 1}
                    selected={selectedResult?.id === entry.createdAt ? selectedResult.item : undefined} />
                </div>}
                {!entry.pending && !entry.error && <button type="button" className="icon-button calculator-line-action calculator-history-actions calculator-copy-result"
                  disabled={entry.result.value !== savedRows[index]?.result.value} aria-label={t("复制结果")} title={t("复制结果")} onClick={() => copy(entry.result.value)}><Icon name="copy" /></button>}
              </div>}
            </li>)}
            {draftOpen && <li className="calculator-draft" data-calculation-row="draft">
              {editing ? <div className="calculator-line">
                <span className="calculator-row-number" aria-hidden="true">{historyRows.length + 1}</span>
                <button type="button" className="calculator-expression-input calculator-draft-entry" data-empty={!draft} data-placeholder={t("输入算式…")}
                  aria-label={t("返回新算式")} disabled={busy}>
                  <Expression value={draft} display={displayToken} operand={token => linkedOperand(token, savedRows)} />
                </button>
              </div> : editor(historyRows.length + 1)}
            </li>}
          </ol>
        </div>
      </div>
    </div>
    <div className="calculator-controls">
      {incoming && !busy && (draft || editingChanged) && <div className="calculator-feedback">
        <span role="status">{t("启动器算式待打开，请先完成或取消当前编辑。")}</span>{" "}
        <button type="button" className="secondary-button" onClick={() => onImportHandled(incoming)}>{t("取消导入")}</button>
      </div>}
      <p className={message?.error || previewError ? "calculator-feedback error" : "calculator-feedback"} role={message?.error || previewError ? "alert" : "status"}>{feedback}</p>
      <div className="calculator-keys" role="group" aria-label={t("计算键盘")}>
        {keys.map(key => {
          const glyph = keyGlyphs[key];
          return <button type="button" key={key} data-key={key}
            disabled={key === "=" && (!native || !expression.trim())}
            className={key === "=" ? "calculator-equals" : "+−×÷".includes(key) ? "calculator-operator" : undefined}
            onMouseDown={event => event.preventDefault()}
            onClick={() => key === "=" ? commit() : press(key)}
            aria-label={key === "AC" ? t("清除算式") : key === "Backspace" ? t("退格") : key === "=" ? t(editing ? "更新计算记录" : "计算并保存") : glyph ? key : undefined}>
            {key === "Backspace" ? <Icon name="backspace" /> : glyph ?
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" aria-hidden="true"><path d={glyph} /></svg> : key}
          </button>;
        })}
      </div>
    </div>
  </form>;
}
