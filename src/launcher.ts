import { locale, onLocaleChange } from "mygo-runtime";
import { System, events, type LauncherResult } from "./mygo";
import en from "../internal/localization/en.json";
import zh from "../internal/localization/zh.json";
import { resolveLanguage } from "./i18n-language";

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
let request = 0;

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

function row(r: LauncherResult, i: number): HTMLLIElement {
  const li = document.createElement("li");
  li.id = `launcher-item-${i}`;
  li.setAttribute("role", "option");
  li.setAttribute("aria-selected", String(i === selected));
  const img = document.createElement("img");
  img.alt = "";
  img.src = iconURL(r);
  img.onerror = () => { img.style.visibility = "hidden"; };
  const copy = document.createElement("div");
  copy.className = "launcher-copy";
  const title = document.createElement("span");
  title.className = "launcher-title";
  title.textContent = r.title;
  const subtitle = document.createElement("span");
  subtitle.className = "launcher-subtitle";
  subtitle.textContent = r.subtitle;
  copy.append(title, subtitle);
  const hint = document.createElement("span");
  hint.className = "launcher-hint";
  hint.textContent = r.kind === "app" ? `↩ ${t("打开")}` : `↩ ${t("打开")}  ${isMac ? "⌘↩" : "Ctrl+↩"} ${t(isMac ? "在访达中显示" : "在文件夹中显示")}`;
  li.append(img, copy, hint);
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
  empty.hidden = !(results.length === 0 && input.value.trim() !== "");
  empty.textContent = t("没有找到匹配的结果");
  input.setAttribute("aria-expanded", String(results.length > 0));
  mark();
  requestAnimationFrame(() => {
    const style = getComputedStyle(document.body);
    void System.launcherResize(Math.ceil(card.getBoundingClientRect().height + parseFloat(style.paddingTop) + parseFloat(style.paddingBottom)));
  });
}

async function search() {
  const id = ++request;
  const found = await System.launcherQuery(input.value);
  if (id !== request) return;
  results = found;
  selected = 0;
  render();
}

async function open(action: "open" | "reveal" | "copy") {
  const r = results[selected];
  if (!r) return;
  try { await System.launcherOpen(r.id, action); }
  catch (reason) { console.error(reason); }
}

input.addEventListener("input", () => { void search(); });
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
    void open(modifier(event) ? "reveal" : "open");
  } else if (modifier(event) && event.key.toLowerCase() === "c" && input.selectionStart === input.selectionEnd && results.length) {
    event.preventDefault(); // with no text selected, ⌘C (Ctrl+C) copies the selected result's path
    void open("copy");
  } else if (event.key === "Escape") {
    event.preventDefault();
    if (input.value) { input.value = ""; void search(); }
    else void System.launcherHide();
  }
});

events.launcherShown.on(() => {
  applyLanguage(locale());
  input.focus();
  input.select();
  void search();
});
onLocaleChange(applyLanguage);
applyLanguage(locale());
