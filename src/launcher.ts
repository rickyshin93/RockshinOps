import { locale, onLocaleChange } from "mygo-runtime";
import { springValue } from "motion";
import { System, events, type LauncherResult } from "./mygo";
import en from "../internal/localization/en.json";
import zh from "../internal/localization/zh.json";
import { resolveLanguage } from "./i18n-language";
import { formatCalculation } from "./calculation-format";

// The panel stays light: plain DOM and the shared catalogs, no React or i18next.
const catalogs: Record<"en" | "zh", Record<string, string>> = { en, zh };
let language: "en" | "zh" = "en";
const t = (key: string) => catalogs[language][key] ?? key;
const isMac = navigator.userAgent.includes("Mac");
const modifier = (event: KeyboardEvent | MouseEvent) => isMac ? event.metaKey : event.ctrlKey;

const input = document.getElementById("launcher-input") as HTMLInputElement;
const list = document.getElementById("launcher-results") as HTMLUListElement;
const resultsScroll = document.getElementById("launcher-results-scroll") as HTMLDivElement;
const highlight = document.getElementById("launcher-highlight") as HTMLDivElement;
const highlightOffset = springValue<number>(0, { stiffness: 2100, damping: 92, restDelta: 0.35, restSpeed: 10 });
highlightOffset.on("change", offset => {
  highlight.style.transform = `translate3d(0, ${offset}%, 0)`;
});
const reduceMotion = matchMedia("(prefers-reduced-motion: reduce)");
const empty = document.getElementById("launcher-empty") as HTMLParagraphElement;
const card = document.getElementById("launcher") as HTMLDivElement;

let results: LauncherResult[] = [];
let selected = 0;
let request = Promise.resolve();
let operation = request;
let displayed: Promise<void>;
let feedback = "";
let opening = false;
let resultRows = new Map<string, { item: LauncherResult; element: HTMLLIElement }>();
let resizeFrame = 0;
let lastHeight = 0;
let waitingForPointer = false;

function applyLanguage(tag: string) {
  const next = resolveLanguage([tag]);
  if (document.documentElement.lang === next && input.placeholder) return;
  resultRows.clear();
  language = next;
  document.documentElement.lang = language;
  input.placeholder = t("搜索应用和文件");
  input.setAttribute("aria-label", input.placeholder);
  render();
}

// Apps draw their own icon; files and folders a generic icon of their type, cached per type.
function iconURL(r: LauncherResult): string {
  if (r.kind === "app") return `appicon://localhost/?path=${encodeURIComponent(r.path)}`;
  if (r.kind === "folder") return "appicon://localhost/type?folder=1";
  const dot = r.title.lastIndexOf(".");
  const ext = dot > 0 ? r.title.slice(dot + 1).toLowerCase() : "";
  return `appicon://localhost/type?ext=${/^[a-z0-9]{1,16}$/.test(ext) ? ext : ""}`;
}

function resultIcon(r: LauncherResult) {
  if (r.kind === "calculation") {
    const icon = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    icon.setAttribute("class", "launcher-result-icon");
    icon.setAttribute("viewBox", "0 0 24 24");
    icon.setAttribute("aria-hidden", "true");
    icon.innerHTML = '<rect x="5" y="2" width="14" height="20" rx="2"/><path d="M8 7h8M9 12h1m4 0h1m-6 4h1m4 0h1"/>';
    return icon;
  }
  const img = document.createElement("img");
  img.alt = "";
  img.src = iconURL(r);
  img.onerror = () => { img.style.visibility = "hidden"; };
  return img;
}

function actionHint(label: string, modified = false): HTMLElement {
  const action = document.createElement("span");
  action.className = "launcher-action";
  const keys = document.createElement("span");
  keys.className = "launcher-keys";
  keys.setAttribute("aria-hidden", "true");
  if (modified) {
    const modifierKey = document.createElement("kbd");
    modifierKey.className = "launcher-keycap";
    modifierKey.textContent = isMac ? "⌘" : "Ctrl";
    keys.append(modifierKey);
  }
  const returnKey = document.createElement("kbd");
  returnKey.className = "launcher-keycap";
  returnKey.innerHTML = '<svg viewBox="0 0 16 16"><path d="M13 4.5v3a2 2 0 0 1-2 2H3m3-3-3 3 3 3"/></svg>';
  keys.append(returnKey);
  action.append(keys, t(label));
  return action;
}

function row(r: LauncherResult): HTMLLIElement {
  const li = document.createElement("li");
  if (r.kind === "calculation") li.className = "launcher-calculation";
  li.setAttribute("role", "option");
  const copy = document.createElement("div");
  copy.className = "launcher-copy";
  const title = document.createElement("span");
  title.className = "launcher-title";
  title.textContent = r.kind === "calculation" ? formatCalculation({ value: r.title, approximate: !!r.approximate }, language, r.grouping) : r.title;
  const subtitle = document.createElement("span");
  subtitle.className = "launcher-subtitle";
  subtitle.textContent = r.subtitle;
  copy.append(title, subtitle);
  const hint = document.createElement("span");
  hint.className = "launcher-hint";
  if (r.kind === "calculation") {
    hint.append(actionHint("复制结果"), actionHint("在计算器中打开", true));
  } else {
    hint.append(actionHint("打开"));
    if (r.kind !== "app") hint.append(actionHint(isMac ? "在访达中显示" : "在文件夹中显示", true));
  }
  li.append(resultIcon(r), copy, hint);
  li.addEventListener("pointerenter", event => {
    if (!waitingForPointer && event.pointerType !== "touch") selectRow(Number(li.dataset.index), true);
  });
  li.addEventListener("click", event => {
    selectRow(Number(li.dataset.index));
    void open(modifier(event) ? "reveal" : "open");
  });
  return li;
}

function markRow(element: Element | null, active: boolean) {
  const value = String(active);
  if (element && element.getAttribute("aria-selected") !== value) element.setAttribute("aria-selected", value);
}

function updateActiveDescendant() {
  input.setAttribute("aria-activedescendant", list.children.item(selected)?.id ?? "");
}

function positionHighlight(slide = false) {
  const offset = selected * 100;
  if (slide && !reduceMotion.matches) {
    highlightOffset.set(offset);
  } else {
    highlightOffset.jump(offset);
  }
}

function selectRow(index: number, pointer = false) {
  if (index === selected) return;
  markRow(list.children.item(selected), false);
  selected = index;
  markRow(list.children.item(selected), true);
  updateActiveDescendant();
  positionHighlight(pointer);
}

function resumePointerSelection(event: PointerEvent) {
  if (event.pointerType === "touch") return;
  waitingForPointer = false;
  list.removeEventListener("pointermove", resumePointerSelection);
  const element = (event.target as Element).closest<HTMLLIElement>("li");
  if (element?.parentElement === list) selectRow(Number(element.dataset.index), true);
}

// A new list or keyboard move can change which row lies under a parked pointer.
// Let actual movement take over, including movement within that same row.
function waitForPointer() {
  if (waitingForPointer) return;
  waitingForPointer = true;
  list.addEventListener("pointermove", resumePointerSelection);
}

function updateRow(element: HTMLLIElement, index: number) {
  if (element.dataset.index !== String(index)) {
    element.dataset.index = String(index);
    element.id = `launcher-item-${index}`;
  }
  markRow(element, index === selected);
}

function renderRows() {
  const previous = resultRows;
  resultRows = new Map();
  const removed = new Set(list.children);
  for (const item of results) {
    const cached = previous.get(item.id);
    const element = cached && cached.item.title === item.title && cached.item.subtitle === item.subtitle
      && cached.item.approximate === item.approximate && cached.item.grouping === item.grouping
      ? cached.element
      : row(item);
    resultRows.set(item.id, { item, element });
    removed.delete(element);
  }
  // Keep matching rows connected when the same results arrive on reopen or filter.
  for (const child of removed) child.remove();
  let index = 0;
  for (const { element } of resultRows.values()) {
    updateRow(element, index);
    const current = list.children.item(index++);
    if (current !== element) list.insertBefore(element, current);
  }
}

function render() {
  renderRows();
  resultsScroll.hidden = results.length === 0;
  positionHighlight();
  empty.hidden = !feedback && !(results.length === 0 && input.value.trim() !== "");
  empty.textContent = t(feedback || "没有找到匹配的结果");
  input.setAttribute("aria-expanded", String(results.length > 0));
  updateActiveDescendant();
  if (resizeFrame) return;
  resizeFrame = requestAnimationFrame(() => {
    resizeFrame = 0;
    const style = getComputedStyle(document.body);
    const height = Math.ceil(card.getBoundingClientRect().height + parseFloat(style.paddingTop) + parseFloat(style.paddingBottom));
    if (height === lastHeight) return;
    lastHeight = height;
    void System.launcherResize(height);
  });
}

function search() {
  feedback = "";
  const query = input.value;
  const next = operation.then(async () => {
    if (next !== request) return;
    const found = await System.launcherQuery(query);
    if (next !== request) return;
    results = found.items;
    if (found.calculatorError) {
      console.error(found.calculatorError);
      feedback = "无法读取计算器设置，应用和文件仍可搜索。";
    }
    selected = 0;
    waitForPointer();
    displayed = next;
    render();
  }).catch(reason => {
    if (next !== request) return;
    console.error(reason);
    results = [];
    feedback = "搜索失败，请重试。";
    render();
  });
  request = next;
  operation = next;
}

async function open(action: "open" | "reveal" | "copy") {
  if (opening || displayed !== request) return;
  const r = results[selected];
  if (!r) return;
  const current = request;
  let resultAction: string = action;
  if (r.kind === "calculation") resultAction = action === "reveal" ? "calculator" : "copy";
  opening = true;
  const actionPromise = System.launcherOpen(r.id, resultAction);
  operation = actionPromise.catch(() => {});
  try { await actionPromise; }
  catch (reason) {
    console.error(reason);
    if (current === request) {
      feedback = r.kind === "calculation" ? "计算器操作失败，请重试。" : "无法打开启动器结果，请重试。";
      render();
    }
  } finally { opening = false; }
}

input.addEventListener("input", search);
input.addEventListener("keydown", event => {
  if (event.isComposing) return;
  if (event.key === "ArrowDown" || event.key === "ArrowUp") {
    event.preventDefault();
    if (!results.length) return;
    waitForPointer();
    selectRow((selected + (event.key === "ArrowDown" ? 1 : results.length - 1)) % results.length);
    list.children[selected]?.scrollIntoView({ block: "nearest" });
  } else if (event.key === "Enter") {
    event.preventDefault();
    const current = request;
    void current.then(() => {
      if (current === request) void open(modifier(event) ? "reveal" : "open");
    });
  } else if (modifier(event) && event.key.toLowerCase() === "c" && input.selectionStart === input.selectionEnd && results.length) {
    event.preventDefault(); // with no text selected, copy the selected result
    void open("copy");
  } else if (event.key === "Escape") {
    event.preventDefault();
    if (input.value) { input.value = ""; search(); }
    else void System.launcherHide();
  }
});

events.launcherShown.on(() => {
  input.focus();
  input.select();
  search();
});
events.launcherHidden.on(() => positionHighlight());
onLocaleChange(applyLanguage);
reduceMotion.addEventListener("change", () => positionHighlight());
applyLanguage(locale());
// The first invocation creates this page; its shown event may precede loading.
input.focus();
search();
