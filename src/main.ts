import { t, i18n, resolveLanguage, type Language } from "./i18n";
import { toggleAllAppFiles, reconcileAppFiles } from "./app-selection";
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
  busy: false, processError: "", connectionError: "", scanError: "", scanning: false, cleanupBusy: false, cleanupMessage: "", spaceMode: false, spaceHistory: [],
  appMode: false, installedApps: null, appReview: null, appSelection: [], appBusy: false, appSearching: false, appMessage: "",
  preferences: null, menuBarDraft: [], languageDraft: "system", preferencesBusy: false, preferencesError: "",
};
let settingsRoot: Root | null = null;
void i18n.changeLanguage(isMyGo() ? locale() : resolveLanguage(navigator.languages));
document.documentElement.lang = i18n.language;
let updating = false;
let processTime = "";
let stopped = false;
let renderedActions = "";
let renderedScan = "";
let renderedAppReview = "";
let scanProgressText = "";
let appReviewEpoch = 0;
let appSelectAll = false;
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
  const focusKey = focusArea ? focused?.dataset.action || focused?.dataset.range || focused?.dataset.page || focused?.dataset.app || focused?.dataset.appFile || focused?.dataset.appSelectAll || focused?.dataset.process || focused?.dataset.metric : undefined;
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
    renderedScan = "";
    renderedAppReview = "";
    if (!settingsRoot) { el("page-body").replaceChildren(); settingsRoot = createRoot(el("page-body")); }
    settingsRoot.render(createElement(I18nextProvider, { i18n }, createElement(SettingsView, { state, onLanguage: (language: Language) => { state.languageDraft = language; render(); } })));
  } else {
    settingsRoot?.unmount(); settingsRoot = null;
    const body = renderBody(state);
    const unchangedScan = state.scanning && state.page === "cleanup" && renderedScan === body;
    const unchangedReview = state.page === "apps" && !!state.appReview && renderedAppReview === body;
    if (!unchangedScan && !unchangedReview) el("page-body").innerHTML = body;
    renderedScan = state.scanning && state.page === "cleanup" ? body : "";
    renderedAppReview = state.page === "apps" && state.appReview ? body : "";
    const progress = el("page-body").querySelector<HTMLElement>("[data-scan-progress]");
    if (progress) progress.textContent = scanProgressText || t("正在统计扫描项目…");
    const selectAll = el("page-body").querySelector<HTMLInputElement>("[data-app-select-all]");
    if (selectAll && state.appReview) selectAll.indeterminate = state.appSelection.length > 0 && state.appSelection.length < state.appReview.files.length;
  }
  el("page-body").setAttribute("aria-busy", String(state.busy || state.scanning));
  if (focusKey) for (const button of focusArea!.querySelectorAll<HTMLElement>("button, input")) {
    if ([button.dataset.action, button.dataset.range, button.dataset.page, button.dataset.app, button.dataset.appFile, button.dataset.appSelectAll, button.dataset.process, button.dataset.metric].includes(focusKey)) { button.focus({ preventScroll: true }); break; }
  }
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
  if (state.scanning || state.cleanupBusy) return;
  state.scanning = true; state.scanError = ""; scanProgressText = ""; render();
  let polling = false, active = true;
  const timer = state.spaceMode ? window.setInterval(async () => {
    if (polling) return;
    polling = true;
    try {
      const progress = await System.scanProgress();
      if (!active) return;
      scanProgressText = progress.phase === "scanning" && progress.total > 0
        ? t("已扫描 {{percent}}%（{{visited}} / {{total}} 项）", { percent: Math.min(99, Math.floor(progress.visited / progress.total * 100)), visited: progress.visited, total: progress.total })
        : progress.phase === "counting" ? t("正在统计扫描项目：已发现 {{visited}} 项", { visited: progress.visited }) : t("正在统计扫描项目…");
      const element = el("page-body").querySelector<HTMLElement>("[data-scan-progress]");
      if (element) element.textContent = scanProgressText;
    } catch {
      if (!active) return;
      scanProgressText = t("无法读取扫描进度，扫描仍在进行。");
      const element = el("page-body").querySelector<HTMLElement>("[data-scan-progress]");
      if (element) element.textContent = scanProgressText;
    } finally { polling = false; }
  }, 500) : undefined;
  try { state.scan = state.spaceMode ? await System.scanDisk(state.spaceHistory.at(-1) || "") : await System.scanCleanup(); }
  catch (reason) { state.scanError = String(reason); }
  finally { active = false; if (timer !== undefined) window.clearInterval(timer); state.scanning = false; render(); }
}
async function cleanup(path: string, emptyTrash: boolean) {
  if (state.cleanupBusy || state.scanning || !state.scan) return;
  const prompt = emptyTrash
    ? t("永久删除废纸篓中的所有内容？此操作不可恢复。路径：{{path}}", { path })
    : t("将此项目移入系统废纸篓？目录内的全部内容都会移动，包括扫描未统计的内容。路径：{{path}}", { path });
  if (!window.confirm(prompt)) return;
  state.cleanupBusy = true; state.cleanupMessage = ""; render();
  try {
    if (emptyTrash) await System.emptyCleanupTrash(path); else await System.trashCleanupItem(path);
    state.cleanupMessage = emptyTrash ? t("废纸篓已清空。") : t("项目已移入废纸篓；清空废纸篓后才会释放空间。");
  } catch (reason) { state.cleanupMessage = `${t("清理未完全完成，请检查后重试：")}${String(reason)}`; }
  finally { state.cleanupBusy = false; await scan(); }
}
async function openFullDiskAccessSettings() {
  try { await System.openFullDiskAccessSettings(); if (errorKind === "action") error("", "action"); }
  catch (reason) { error(t("无法打开系统设置：{{error}}", { error: String(reason) }), "action"); }
}
async function installedApps() {
  if (state.appBusy) return;
  state.appMode = true; state.appReview = null; state.appSelection = []; state.appBusy = true; state.appMessage = ""; render(true);
  try { state.installedApps = await System.installedApps(); }
  catch (reason) { state.installedApps = null; state.appMessage = String(reason); }
  finally { state.appBusy = false; render(); }
}
async function reviewApp(path: string) {
  if (state.appBusy) return;
  const epoch = ++appReviewEpoch;
  appSelectAll = false;
  state.appBusy = true; state.appSearching = true; state.appReview = null; state.appMessage = ""; state.appSelection = []; render();
  try {
    const quick = await System.reviewAppQuick(path);
    if (epoch !== appReviewEpoch) { await System.cancelAppReview(quick.reviewId); return; }
    state.appReview = quick; render(true);
    const review = await System.reviewApp(path, state.appReview.reviewId);
    if (epoch !== appReviewEpoch) return;
    state.appReview = review;
    state.appSelection = reconcileAppFiles(state.appReview.files.map(file => file.path), state.appSelection, appSelectAll);
  }
  catch (reason) { if (epoch === appReviewEpoch) state.appMessage = String(reason); }
  finally { if (epoch === appReviewEpoch) { state.appBusy = false; state.appSearching = false; render(true); } }
}
async function stopAppSearch() {
  if (!state.appSearching || !state.appReview) return;
  try {
    await System.cancelAppReview(state.appReview.reviewId);
    if (!state.appSearching) return;
    appReviewEpoch++;
    state.appBusy = false;
    state.appSearching = false;
    state.appMessage = t("已停止搜索，仅保留常见位置的首批项目。");
    render();
  } catch (reason) { state.appMessage = String(reason); render(); }
}
async function removeAppFiles() {
  if (state.appBusy || !state.appReview || !state.appSelection.length) return;
  if (!window.confirm(`${t("将以下选中项目移入系统废纸篓？请确认 App 已退出。") }\n\n${state.appSelection.join("\n")}`)) return;
  const path = state.appReview.app.path;
  state.appBusy = true; state.appMessage = ""; render();
  try { await System.removeAppFiles([...state.appSelection]); state.appMessage = t("选中项目已移入废纸篓。"); }
  catch (reason) { state.appMessage = `${t("清理未完全完成，请检查后重试：")}${String(reason)}`; }
  state.appSelection = []; state.appReview = null;
  try { state.installedApps = await System.installedApps(); }
  catch (reason) { state.appMessage += ` ${String(reason)}`; state.installedApps = null; }
  state.appBusy = false; render();
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
  if (button.dataset.cleanupPath) { void cleanup(button.dataset.cleanupPath, false); return; }
  if (button.dataset.emptyTrash) { void cleanup(button.dataset.emptyTrash, true); return; }
  if (button.dataset.spacePath) { state.spaceHistory.push(button.dataset.spacePath); void scan(); return; }
  if (button.dataset.reviewApp) { void reviewApp(button.dataset.reviewApp); return; }
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
    case "space-start": state.spaceMode = true; state.spaceHistory = []; state.scan = null; void scan(); break;
    case "space-back": state.spaceHistory.pop(); void scan(); break;
    case "cleanup-targets": state.spaceMode = false; state.spaceHistory = []; state.scan = null; void scan(); break;
    case "installed-apps": void installedApps(); break;
    case "open-full-disk-access": void openFullDiskAccessSettings(); break;
    case "running-apps":
      if (state.appSearching && state.appReview) void stopAppSearch();
      else if (state.appSearching) { appReviewEpoch++; state.appBusy = false; state.appSearching = false; }
      state.appMode = false; state.appReview = null; render(true); break;
    case "remove-app-files": void removeAppFiles(); break;
    case "stop-app-search": void stopAppSearch(); break;
    case "hide": void hide(); break;
    case "load-settings": void loadSettings(); break;
    case "save-settings": void saveSettings(); break;
    case "check-updates": void System.checkForUpdates().catch(reason => error(`${t("无法打开更新窗口：")}${String(reason)}`, "action")); break;
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
el("page-body").addEventListener("change", event => {
  const input = event.target as HTMLInputElement;
  if (input.dataset.appSelectAll && state.appReview) {
    appSelectAll = input.checked;
    state.appSelection = toggleAllAppFiles(state.appReview.files.map(file => file.path), !input.checked);
    render(); return;
  }
  if (!input.dataset.appFile) return;
  const selected = new Set(state.appSelection);
  if (input.checked) selected.add(input.dataset.appFile); else selected.delete(input.dataset.appFile);
  state.appSelection = [...selected];
  appSelectAll = !!state.appReview?.files.length && state.appSelection.length === state.appReview.files.length;
  render();
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
