import { locale, onLocaleChange } from "mygo-runtime";
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
const empty = document.getElementById("launcher-empty") as HTMLParagraphElement;
const card = document.getElementById("launcher") as HTMLDivElement;

let results: LauncherResult[] = [];
let selected = 0;
let request = Promise.resolve();
let operation = request;
let displayed: Promise<void>;
let feedback = "";
let opening = false;

function applyLanguage(tag: string) {
  language = resolveLanguage([tag]);
  document.documentElement.lang = language;
  input.placeholder = t("搜索应用和文件");
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
    icon.setAttribute("class", "launcher-calculator");
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

function row(r: LauncherResult, i: number): HTMLLIElement {
  const li = document.createElement("li");
  li.id = `launcher-item-${i}`;
  if (r.kind === "calculation") li.className = "launcher-calculation";
  li.setAttribute("role", "option");
  li.setAttribute("aria-selected", String(i === selected));
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
  hint.textContent = `↩ ${t("打开")}`;
  if (r.kind === "calculation") hint.textContent = `↩ ${t("复制结果")}  ${isMac ? "⌘↩" : "Ctrl+↩"} ${t("在计算器中打开")}`;
  else if (r.kind !== "app") hint.textContent += `  ${isMac ? "⌘↩" : "Ctrl+↩"} ${t(isMac ? "在访达中显示" : "在文件夹中显示")}`;
  li.append(resultIcon(r), copy, hint);
  li.addEventListener("mousemove", () => { if (selected !== i) { selected = i; mark(); } });
  li.addEventListener("click", event => { selected = i; void open(modifier(event) ? "reveal" : "open"); });
  return li;
}

function mark() {
  list.querySelectorAll("li").forEach((li, i) => li.setAttribute("aria-selected", String(i === selected)));
  input.setAttribute("aria-activedescendant", results.length ? `launcher-item-${selected}` : "");
}

function render() {
  list.replaceChildren(...results.map(row));
  list.hidden = results.length === 0;
  empty.hidden = !feedback && !(results.length === 0 && input.value.trim() !== "");
  empty.textContent = t(feedback || "没有找到匹配的结果");
  input.setAttribute("aria-expanded", String(results.length > 0));
  mark();
  requestAnimationFrame(() => {
    const style = getComputedStyle(document.body);
    void System.launcherResize(Math.ceil(card.getBoundingClientRect().height + parseFloat(style.paddingTop) + parseFloat(style.paddingBottom)));
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
    selected = (selected + (event.key === "ArrowDown" ? 1 : results.length - 1)) % results.length;
    mark();
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
  applyLanguage(locale());
  input.focus();
  input.select();
  search();
});
onLocaleChange(applyLanguage);
applyLanguage(locale());
// The first invocation creates this page; its shown event may precede loading.
input.focus();
search();
