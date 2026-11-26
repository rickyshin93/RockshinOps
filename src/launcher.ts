import { locale, onLocaleChange } from "mygo-runtime";
import { springValue } from "motion";
import { System, events, type LauncherResult } from "./mygo";
import en from "../internal/localization/en.json";
import zh from "../internal/localization/zh.json";
import { resolveLanguage } from "./i18n-language";
import { formatCalculation } from "./calculation-format";
import { clipboardImageURL, clipboardWarningKey } from "./clipboard-ui";

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
const modeButton = document.getElementById("launcher-mode") as HTMLButtonElement;
const clipboardControls = document.getElementById("launcher-clipboard-controls") as HTMLDivElement;
const clearButton = document.getElementById("launcher-clear") as HTMLButtonElement;

let results: LauncherResult[] = [];
let selected = 0;
let request = Promise.resolve();
let operation = request;
let displayed: Promise<void>;
let feedback = "";
let opening: { appLaunch: boolean } | null = null;
let clipboardMode = false;
let clipboardStatus: string | undefined;
let clipboardCount = 0;
let clipboardPinnedCount = 0;
let clipboardWarning = "";
let clipboardBusy = false;
let visible = true;
let presentation = 0;
let resultRows = new Map<string, { item: LauncherResult; element: HTMLLIElement }>();
let resizeFrame = 0;
let lastHeight = 0;
let waitingForPointer = false;

function applyLanguage(tag: string) {
  const next = resolveLanguage([tag]);
  if (language === next) return;
  resultRows.clear();
  language = next;
  document.documentElement.lang = language;
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
  if (r.clipboardImage) {
    const image = document.createElement("img");
    image.className = "launcher-image";
    image.alt = "";
    image.src = clipboardImageURL(r.id);
    return image;
  }
  if (r.kind === "calculation" || r.kind === "clipboard") {
    const icon = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    icon.setAttribute("class", "launcher-result-icon");
    icon.setAttribute("viewBox", "0 0 24 24");
    icon.setAttribute("aria-hidden", "true");
    icon.innerHTML = r.kind === "clipboard"
      ? '<rect x="8" y="2" width="8" height="4" rx="1"/><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2m0 7h8m-8 5h8"/>'
      : '<rect x="5" y="2" width="14" height="20" rx="2"/><path d="M8 7h8M9 12h1m4 0h1m-6 4h1m4 0h1"/>';
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
  if (r.kind === "clipboard") li.className = "launcher-clipboard";
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
  } else if (r.kind === "clipboard") {
    hint.append(actionHint("复制"));
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
  card.dataset.clipboard = String(clipboardMode);
  input.placeholder = t(clipboardMode ? "搜索剪贴板历史" : "搜索应用和文件");
  input.setAttribute("aria-label", input.placeholder);
  modeButton.hidden = !isMac;
  modeButton.textContent = t(clipboardMode ? "应用和文件" : "剪贴板");
  modeButton.title = `${t(clipboardMode ? "搜索应用和文件" : "剪贴板历史")} (⌘4)`;
  clipboardControls.hidden = !clipboardMode || !clipboardStatus;
  clearButton.textContent = t(clipboardPinnedCount ? "清空未置顶历史" : "清空历史");
  clearButton.disabled = clipboardBusy || clipboardCount <= clipboardPinnedCount;
  renderRows();
  resultsScroll.hidden = results.length === 0;
  positionHighlight();
  let message = feedback;
  if (!message && clipboardMode) {
    if (clipboardStatus === "permission") message = "请在 macOS 系统设置中始终允许 RockshinOps 访问剪贴板。";
    else if (clipboardWarning) message = clipboardWarning;
    else if (clipboardStatus && results.length === 0) {
      message = input.value.trim() ? "没有找到匹配的结果" : "复制文本或图片，即可在这里找回。";
    }
  } else if (!message && results.length === 0 && input.value.trim()) {
    message = "没有找到匹配的结果";
  }
  empty.hidden = !message;
  empty.textContent = message ? t(message) : "";
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

function search(preserveSelection = false) {
  feedback = "";
  const query = input.value;
  const clipboard = clipboardMode;
  const next = operation.then(async () => {
    if (next !== request) return;
    const found = await (clipboard ? System.launcherClipboardQuery(query) : System.launcherQuery(query));
    if (next !== request) return;
    const selectedId = preserveSelection ? results[selected]?.id : undefined;
    results = found.items;
    clipboardStatus = found.clipboardStatus;
    clipboardCount = found.clipboardCount ?? 0;
    clipboardPinnedCount = found.clipboardPinnedCount ?? 0;
    clipboardWarning = clipboardWarningKey(found.clipboardWarning);
    if (found.calculatorError) {
      console.error(found.calculatorError);
      feedback = "无法读取计算器设置，应用和文件仍可搜索。";
    }
    selected = Math.max(0, selectedId ? results.findIndex(item => item.id === selectedId) : 0);
    waitForPointer();
    displayed = next;
    render();
    if (preserveSelection && selected > 0) list.children[selected]?.scrollIntoView({ block: "nearest" });
    else resultsScroll.scrollTop = 0;
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
  if (!r || (r.kind === "clipboard" && clipboardBusy)) return;
  const current = request;
  let resultAction: string = action;
  if (r.kind === "calculation") resultAction = action === "reveal" ? "calculator" : "copy";
  if (r.kind === "clipboard") resultAction = "copy";
  const pending = { appLaunch: isMac && r.kind === "app" && resultAction === "open" };
  opening = pending;
  const actionPromise = System.launcherOpen(r.id, resultAction);
  if (!pending.appLaunch) operation = actionPromise.catch(() => {});
  try { await actionPromise; }
  catch (reason) {
    console.error(reason);
    if (opening === pending && current === request) {
      feedback = r.kind === "calculation" ? "计算器操作失败，请重试。" : r.kind === "clipboard" ? "无法复制这条记录，请重新搜索。" : "无法打开启动器结果，请重试。";
      render();
    }
  } finally { if (opening === pending) opening = null; }
}

function resetClipboardView() {
  request = Promise.resolve();
  results = [];
  selected = 0;
  feedback = "";
  clipboardStatus = undefined;
  clipboardCount = 0;
  clipboardPinnedCount = 0;
  clipboardWarning = "";
  input.value = "";
}

function setMode(clipboard: boolean) {
  if (clipboard !== clipboardMode) {
    resetClipboardView();
    clipboardMode = clipboard;
    render();
  }
  input.focus();
  input.select();
  search();
}

modeButton.addEventListener("click", () => setMode(!clipboardMode));
async function clearClipboard() {
  if (clipboardBusy) return;
  clipboardBusy = true;
  render();
  const shown = presentation;
  const change = operation.then(() => System.clearClipboardHistory());
  operation = change.catch(() => {});
  try {
    await change;
    if (shown === presentation && clipboardMode) { resetClipboardView(); input.focus(); search(); render(); }
  } catch (reason) {
    console.error(reason);
    if (shown === presentation && clipboardMode) feedback = "无法清空剪贴板历史。";
  } finally { clipboardBusy = false; render(); }
}
clearButton.addEventListener("click", () => { void clearClipboard(); });

input.addEventListener("input", () => search());
document.addEventListener("keydown", event => {
  if (isMac && !event.isComposing && event.metaKey && !event.altKey && !event.ctrlKey && !event.shiftKey && event.key === "4") {
    event.preventDefault();
    if (!event.repeat) setMode(!clipboardMode);
  }
});
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

async function show(clipboard?: boolean) {
  const shown = ++presentation;
  visible = true;
  if (opening?.appLaunch) opening = null;
  if (clipboard !== undefined || !isMac) {
    setMode(clipboard ?? false);
    return;
  }
  input.focus();
  try {
    const clipboard = await System.launcherShowingClipboard();
    if (shown === presentation) setMode(clipboard);
  } catch (reason) {
    console.error(reason);
    if (shown === presentation) setMode(false);
  }
}

events.launcherShown.on(clipboard => { void show(clipboard); });
events.launcherHidden.on(() => {
  presentation++;
  visible = false;
  positionHighlight();
  if (clipboardMode) { resetClipboardView(); render(); }
});
events.clipboardHistoryChanged.on(() => {
  if (visible && clipboardMode && !clipboardBusy) search(true);
});
events.launcherAppsChanged.on(() => {
  if (visible && !clipboardMode && input.value.trim()) search(true);
});
onLocaleChange(applyLanguage);
reduceMotion.addEventListener("change", () => positionHighlight());
applyLanguage(locale());
// The first invocation creates this page; its shown event may precede loading.
void show();
