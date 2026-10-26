import { t, i18n, resolveLanguage, type Language } from "./i18n";
import { createRoot, type Root } from "react-dom/client";
import { createElement } from "react";
import { I18nextProvider } from "react-i18next";
import { SettingsView } from "./settings-view";
import { isMyGo, locale, onLocaleChange } from "mygo-runtime";
import { System, events, type Snapshot } from "./mygo";
import { formatCapacity, percent, timeLabel } from "./format";
import { icon } from "./icons";
import { installSortControl } from "./dropdown";
import { menuBarChoices, preferencesDirty } from "./settings";
import { actionControls, filterControls, metricPages, navigation, pages, renderBody, type Page, type ViewState } from "./views";

const el = (id: string) => document.getElementById(id)!;
const state: ViewState = {
  page: "overview", info: null, snapshot: null, history: [], range: 60, paused: false,
  processes: null, connections: null, scan: null, query: "", sort: "cpu", onlyMine: false,
  busy: false, processError: "", connectionError: "", scanError: "", scanning: false,
  preferences: null, menuBarDraft: [], languageDraft: "system", preferencesBusy: false, preferencesError: "",
};
let settingsRoot: Root | null = null;
void i18n.changeLanguage(isMyGo() ? locale() : resolveLanguage(navigator.languages));
document.documentElement.lang = i18n.language;
let updating = false;
let processTime = "";
let stopped = false;
let renderedActions = "";
let errorKind: "data" | "action" = "data";
el("navigation").innerHTML = navigation();
el("device-mark").innerHTML = icon("about");

function error(message: string, kind: "data" | "action" = "data") {
  errorKind = kind;
  el("connection-error").textContent = message;
  el("connection-error").hidden = !message;
}
function render(filters = false) {
  const page = pages.find(p => p.id === state.page)!;
  const focused = document.activeElement as HTMLElement | null;
  const focusArea = focused?.closest("#page-actions, #page-body");
  const focusKey = focusArea ? focused?.dataset.action || focused?.dataset.range || focused?.dataset.page || focused?.dataset.app || focused?.dataset.process || focused?.dataset.metric : undefined;
  el("page-title").textContent = t(page.title);
  el("page-subtitle").textContent = state.snapshot ? `${!el("connection-error").hidden && errorKind === "data" ? t("更新中断 · 上次数据") : state.paused ? t("视图已暂停") : t("本机实时数据")} · ${timeLabel(state.snapshot.updatedAt)}` : t("等待本机数据");
  if (state.page === "settings") el("page-subtitle").textContent = t("让菜单栏只显示你关心的信息");
  const actions = actionControls(state);
  if (renderedActions !== actions) { el("page-actions").innerHTML = actions; renderedActions = actions; }
  if (filters) {
    const html = filterControls(state);
    el("page-filters").innerHTML = html;
    el("page-filters").hidden = !html;
  }
  if (state.page === "settings") {
    if (!settingsRoot) { el("page-body").replaceChildren(); settingsRoot = createRoot(el("page-body")); }
    settingsRoot.render(createElement(I18nextProvider, { i18n }, createElement(SettingsView, { state, onLanguage: (language: Language) => { state.languageDraft = language; render(); } })));
  } else {
    settingsRoot?.unmount(); settingsRoot = null;
    el("page-body").innerHTML = renderBody(state);
  }
  el("page-body").setAttribute("aria-busy", String(state.busy || state.scanning));
  if (focusKey) for (const button of focusArea!.querySelectorAll<HTMLElement>("button")) {
    if ([button.dataset.action, button.dataset.range, button.dataset.page, button.dataset.app, button.dataset.process, button.dataset.metric].includes(focusKey)) { button.focus({ preventScroll: true }); break; }
  }
  el("page-footer").textContent = state.info ? t("运行摘要", { version: state.info.version, metrics: state.info.refreshSeconds, processes: state.info.processRefreshSeconds, hours: state.info.historySeconds / 3600 }) : t("LocalOps · 数据仅在本机读取");
  for (const button of el("navigation").querySelectorAll<HTMLButtonElement>("button")) {
    if (button.dataset.page === state.page) button.setAttribute("aria-current", "page");
    else button.removeAttribute("aria-current");
  }
  const s = state.snapshot;
  if (s) {
    el("nav-cpu").textContent = s.cpu.ready ? percent(s.cpu.usage) : "—";
    el("nav-memory").textContent = !s.memory.error ? percent(s.memory.used / s.memory.total * 100) : "—";
    el("nav-disk").textContent = s.disk.capacityError ? "—" : formatCapacity(s.disk.volume.available);
  }
}
async function loadProcesses() {
  try { state.processes = await System.processes(); state.processError = state.processes.error; }
  catch (reason) { state.processError = String(reason); }
  processTime = state.processes?.updatedAt || "";
}
async function receive(snapshot: Snapshot) {
  if (stopped || state.paused || !state.info || updating) return;
  if (state.snapshot && Date.parse(snapshot.updatedAt) < Date.parse(state.snapshot.updatedAt)) return;
  updating = true;
  try {
    const history = await System.history();
    if (state.paused || stopped) return;
    if (!state.snapshot || Date.parse(snapshot.updatedAt) >= Date.parse(state.snapshot.updatedAt)) {
      state.snapshot = snapshot;
      state.history = history.filter(p => Date.parse(p.at) <= Date.parse(snapshot.updatedAt));
    }
    if (["processes", "apps"].includes(state.page) && Date.parse(snapshot.updatedAt) - Date.parse(processTime || "1970-01-01") >= state.info.processRefreshSeconds * 1000) await loadProcesses();
    if (errorKind === "data") error("");
    render();
  } catch (reason) { error(`${t("读取历史失败：")}${String(reason)}${t("。当前显示上次成功的数据。")}`); render(); }
  finally { updating = false; }
}
async function initialize() {
  const result = await System.initialize();
  state.info = result.info;
  state.snapshot = result.snapshot;
  state.history = result.history;
  el("machine-name").textContent = t("本机");
  el("machine-name").title = result.info.machine.hostname;
  el("machine-detail").textContent = result.info.machine.chip || result.info.machine.architecture;
}
async function scan() {
  if (state.scanning) return;
  state.scanning = true; state.scanError = ""; render();
  try { state.scan = await System.scanCleanup(); }
  catch (reason) { state.scanError = String(reason); }
  finally { state.scanning = false; render(); }
}
async function connections() {
  try { state.connections = await System.connections(); state.connectionError = ""; }
  catch (reason) { state.connectionError = String(reason); }
}
async function refresh() {
  if (state.busy) return;
  if (!isMyGo()) { error(t("实时数据仅在 LocalOps 桌面应用中可用。请运行 bun run dev 启动应用。")); return; }
  state.busy = true; error(""); render();
  try {
    if (!state.info) await initialize();
    else {
      const snapshot = await System.refresh();
      const history = await System.history();
      if (!state.snapshot || Date.parse(snapshot.updatedAt) >= Date.parse(state.snapshot.updatedAt)) {
        state.snapshot = snapshot; state.history = history.filter(p => Date.parse(p.at) <= Date.parse(snapshot.updatedAt));
      }
    }
    if (["processes", "apps"].includes(state.page)) await loadProcesses();
    if (state.page === "connections") await connections();
    if (state.page === "cleanup") await scan();
  } catch (reason) { error(`${t("连接采集服务失败：")}${String(reason)}${t("。可再次点击刷新重试。")}`); }
  finally { state.busy = false; render(); }
}
async function navigate(page: Page, query = "") {
  state.page = page; state.query = query;
  if (!metricPages.includes(page)) state.paused = false;
  if (page === "apps" && !["cpu", "memory", "name"].includes(state.sort)) state.sort = "cpu";
  el("content-scroll").scrollTop = 0; render(true);
  if (page === "settings" && !state.preferences) await loadSettings();
  if (!isMyGo() || !state.info) return;
  if (["processes", "apps"].includes(page)) { await loadProcesses(); render(); }
  if (page === "connections" && !state.connections) { await connections(); render(); }
}
async function loadSettings() {
  try {
    state.preferences = await System.preferences();
    state.menuBarDraft = [...state.preferences.menuBarItems]; state.languageDraft = state.preferences.language as Language; state.preferencesError = "";
  } catch (reason) { state.preferencesError = `${t("无法读取设置：")}${String(reason)}`; }
  render();
}
async function saveSettings() {
  if (state.preferencesBusy || !preferencesDirty(state)) return;
  state.preferencesBusy = true; state.preferencesError = ""; render();
  try { state.preferences = await System.savePreferences({ menuBarItems: [...state.menuBarDraft], language: state.languageDraft }); }
  catch (reason) { state.preferencesError = `${t("保存失败，当前菜单栏保持原设置：")}${String(reason)}${t("。请重试。")}`; }
  finally { state.preferencesBusy = false; render(); }
}
async function hide() {
  try { await System.hideWindow(); }
  catch (reason) { error(`${t("无法收起窗口：")}${String(reason)}`, "action"); }
}
document.addEventListener("click", event => {
  const button = (event.target as Element).closest<HTMLButtonElement>("button");
  if (!button || button.disabled) return;
  if (button.dataset.metric) {
    if (state.preferencesBusy) return;
    const selected = new Set(state.menuBarDraft);
    if (selected.has(button.dataset.metric)) selected.delete(button.dataset.metric); else selected.add(button.dataset.metric);
    state.menuBarDraft = menuBarChoices.map(([id]) => id).filter(id => selected.has(id)); render(); return;
  }
  if (button.dataset.page) { void navigate(button.dataset.page as Page); return; }
  if (button.dataset.app) { void navigate("processes", button.dataset.app); return; }
  if (button.dataset.process) { void navigate("processes", button.dataset.process); return; }
  if (button.dataset.range) { state.range = Number(button.dataset.range); render(); return; }
  switch (button.dataset.action) {
    case "pause": state.paused = !state.paused; render(); if (!state.paused) void refresh(); break;
    case "refresh": void refresh(); break;
    case "scan": void scan(); break;
    case "hide": void hide(); break;
    case "load-settings": void loadSettings(); break;
    case "save-settings": void saveSettings(); break;
  }
});
el("page-filters").addEventListener("input", event => {
  const input = event.target as HTMLInputElement;
  if (input.id === "search") { state.query = input.value; render(); }
});
el("page-filters").addEventListener("change", event => {
  const input = event.target as HTMLInputElement;
  if (input.id === "only-mine") state.onlyMine = input.checked;
  render();
});
installSortControl(value => { state.sort = value; render(true); });
document.addEventListener("keydown", event => {
  if ((event.metaKey || event.ctrlKey) && event.key === ",") { event.preventDefault(); void navigate("settings"); }
});
document.querySelector(".sidebar")!.setAttribute("aria-label", t("LocalOps 导航"));
el("navigation").setAttribute("aria-label", t("监控页面"));
el("machine-detail").textContent = t("正在连接本机…");
render(true);
if (isMyGo()) {
  const offLocale = onLocaleChange(language => {
    void i18n.changeLanguage(language);
    document.documentElement.lang = language;
    el("navigation").innerHTML = navigation();
    el("machine-name").textContent = t("本机");
    document.querySelector(".sidebar")!.setAttribute("aria-label", t("LocalOps 导航"));
    el("navigation").setAttribute("aria-label", t("监控页面"));
    state.info = null; state.processes = null; state.connections = null; state.scan = null;
    state.processError = ""; state.connectionError = ""; state.scanError = "";
    error(""); render(true); void refresh();
  });
  const off = events.metricsChanged.on(snapshot => { void receive(snapshot); });
  const offSettings = events.settingsOpened.on(() => { void navigate("settings"); });
  window.addEventListener("pagehide", () => { stopped = true; off(); offSettings(); offLocale(); }, { once: true });
  void refresh();
} else {
  el("machine-detail").textContent = t("浏览器预览");
  error(t("实时数据仅在 LocalOps 桌面应用中可用。请运行 bun run dev 启动应用。"));
}
