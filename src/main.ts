import { t, i18n, resolveLanguage, type Language } from "./i18n";
import { toggleAllAppFiles, reconcileAppFiles } from "./app-selection";
import { isMyGo, locale, onLocaleChange } from "mygo-runtime";
import { System, events, type Preferences, type Snapshot } from "./mygo";
import { formatCapacity, formatRate, percent, timeLabel } from "./format";
import { icon } from "./icons";
import { mergeHistory } from "./history";
import { updateLiveContent } from "./live-content";
import { PageContent } from "./page-content";
import { tableWindow } from "./table-window";
import { singleFlight } from "./single-flight";
import { installSortControl } from "./dropdown";
import { nextConnectionSort, type ConnectionSortKey } from "./connections";
import { listeningPorts } from "./ports";
import { menuBarChoices, preferencesDirty } from "./settings";
import { actionControls, connectionSubtitle, processSubtitle, filterControls, metricPages, navigation, pages, renderBody, type Page, type ViewState } from "./views";

const el = (id: string) => document.getElementById(id)!;
const state: ViewState = {
  page: "overview", info: null, snapshot: null, history: [], range: 60, paused: false,
  processes: null, connections: null, connectionSort: null, connectionMode: "all", portScope: "all", favoritePorts: [], portsLoaded: false, selectedProcess: null, processOrder: { key: "cpu", descending: true }, query: "", sort: "cpu", onlyMine: false,
  busy: false, processError: "", connectionError: "",
  appMode: false, installedApps: null, appReview: null, appSelection: [], appBusy: false, appSearching: false, appMessage: "",
  startup: null, startupBusy: false, startupError: "",
  preferences: null, hideDockDraft: false, updateProxyDraft: false, menuBarDraft: [], trayPreview: null, languageDraft: "system", preferencesBusy: false, preferencesError: "",
};
const reactPage = (page: Page) => ["settings", "menubar", "cleanup"].includes(page);
let reactPages: typeof import("./react-pages") | null = null;
let reactPageError = "";
const loadReactPages = singleFlight(async () => {
  if (reactPages || reactPageError) return;
  try { reactPages = await import("./react-pages"); }
  catch (reason) { reactPageError = String(reason); }
  if (reactPage(state.page)) render();
});
void i18n.changeLanguage(isMyGo() ? locale() : resolveLanguage(navigator.languages));
document.documentElement.lang = i18n.language;
let updating = false;
let queuedSnapshot: Snapshot | null = null;

let connectionTime = 0;
const readConnections = singleFlight(() => System.connections());
const readProcesses = singleFlight(() => System.processes());
let inspectionPreloadScheduled = false;
let stopped = false;
let renderedActions = "";
let renderedAppReview = "";
let renderedAboutLanguage = "";
let renderedAboutInfo: ViewState["info"] = null;
let appReviewEpoch = 0;
let appSelectAll = false;
let navigationEpoch = 0;
let navigationTarget: { page: Page; query: string } | null = null;
let renderedBody = "";
let renderedPage: Page | null = null;
let renderFrame = 0;
let renderFilters = false;
const pageContent = new PageContent(el("page-body"));
let errorKind: "data" | "action" = "data";
el("navigation").innerHTML = navigation();
renderNavigation(state.page);
el("device-mark").innerHTML = icon("about");

function error(message: string, kind: "data" | "action" = "data") {
  errorKind = kind;
  el("connection-error").textContent = message;
  el("connection-error").hidden = !message;
}
function render(filters = false) {
  renderFilters ||= filters;
  if (renderFrame) return;
  renderFrame = requestAnimationFrame(() => {
    renderFrame = 0;
    if (navigationTarget) return;
    const filters = renderFilters;
    renderFilters = false;
    renderNow(filters);
  });
}
function renderNow(filters = false) {
  if (navigationTarget) return;
  if (pageContent.page !== state.page) {
    reactPages?.unmountReactPage();
    renderedBody = pageContent.activate(state.page, renderedBody);
    renderedPage = renderedBody ? state.page : null;
  }
  const page = pages.find(p => p.id === state.page)!;
  const activityScroll = el("page-body").querySelector<HTMLElement>(".activity-scroll");
  const activityPosition = activityScroll ? [activityScroll.scrollLeft, activityScroll.scrollTop] : null;
  const focused = document.activeElement as HTMLElement | null;
  const focusArea = focused?.closest("#page-actions, #page-filters, #page-body");
  const focusKey = focusArea ? focused?.dataset.action || focused?.dataset.range || focused?.dataset.page || focused?.dataset.app || focused?.dataset.appFile || focused?.dataset.appSelectAll || focused?.dataset.process || focused?.dataset.selectProcess || focused?.dataset.metric || focused?.dataset.connectionSort || focused?.dataset.processSort || focused?.dataset.processScope || focused?.dataset.connectionMode || focused?.dataset.favoritePort || focused?.dataset.copyPort || focused?.dataset.portProcess || focused?.dataset.closePort || focused?.dataset.portScope || focused?.dataset.portQuery : undefined;
  const title = t(page.title);
  let subtitle = state.snapshot ? `${!el("connection-error").hidden && errorKind === "data" ? t("更新中断 · 上次数据") : state.paused ? t("视图已暂停") : t("本机实时数据")} · ${timeLabel(state.snapshot.updatedAt)}` : t("等待本机数据");
  if (state.page === "settings") subtitle = t("更改会自动保存");
  if (state.page === "menubar") subtitle = t("管理系统菜单栏图标、分区与外观");
  if (state.page === "about" && state.info) subtitle = state.info.machine.name || state.info.machine.model;
  if (state.page === "connections") subtitle = connectionSubtitle(state);
  if (state.page === "processes") subtitle = processSubtitle(state);
  if (el("page-title").textContent !== title) el("page-title").textContent = title;
  if (el("page-subtitle").textContent !== subtitle) el("page-subtitle").textContent = subtitle;
  if (el("content-scroll").dataset.page !== state.page) {
    el("content-scroll").dataset.page = state.page;
    el("content-scroll").parentElement!.dataset.page = state.page;
  }
  if (state.page === "processes" || state.page === "connections") {
    state.tableWindow = tableWindow(activityScroll?.scrollTop ?? 0, activityScroll?.clientHeight || el("page-body").clientHeight);
  }
  const actions = actionControls(state);
  if (renderedActions !== actions) { el("page-actions").innerHTML = actions; renderedActions = actions; }
  if (filters) {
    const html = filterControls(state);
    el("page-filters").innerHTML = html;
    el("page-filters").hidden = !html;
  }
  if (reactPage(state.page)) {
    renderedBody = "";
    renderedPage = null;
    renderedAppReview = "";
    if (reactPages) {
      reactPages.renderReactPage(el("page-body"), state, (language: Language) => { state.languageDraft = language; render(); void saveSettings(); }, () => void openFullDiskAccessSettings());
    } else {
      el("page-body").textContent = reactPageError ? `${t("页面加载失败，请重试。")} ${reactPageError}` : t("正在加载页面…");
      void loadReactPages();
    }
  } else {
    reactPages?.unmountReactPage();
    const body = renderBody(state);
    const unchangedReview = state.page === "apps" && !!state.appReview && renderedAppReview === body;
    const about = el("page-body").querySelector(".about-page");
    if (state.page === "about" && about && renderedAboutLanguage === i18n.language && renderedAboutInfo === state.info && body.includes('class="about-page"')) {
      const template = document.createElement("template");
      template.innerHTML = body;
      const currentValues = about.querySelectorAll(".about-status span, .about-section dt, .about-section dd");
      const nextValues = template.content.querySelectorAll(".about-status span, .about-section dt, .about-section dd");
      if (currentValues.length === nextValues.length) {
        currentValues.forEach((value, index) => {
          if (value.innerHTML !== nextValues[index]!.innerHTML) value.innerHTML = nextValues[index]!.innerHTML;
        });
      } else el("page-body").innerHTML = body;
    } else if (!unchangedReview && body !== renderedBody) {
      const live = metricPages.includes(state.page) || ["processes", "connections"].includes(state.page) || (state.page === "apps" && !state.appMode);
      if (live && renderedPage === state.page) updateLiveContent(el("page-body"), body);
      else el("page-body").innerHTML = body;
    }
    renderedPage = state.page;
    renderedAboutLanguage = state.page === "about" ? i18n.language : "";
    renderedAboutInfo = state.page === "about" ? state.info : null;
    renderedAppReview = state.page === "apps" && state.appReview ? body : "";
    renderedBody = body;
    const selectAll = el("page-body").querySelector<HTMLInputElement>("[data-app-select-all]");
    if (selectAll && state.appReview) selectAll.indeterminate = state.appSelection.length > 0 && state.appSelection.length < state.appReview.files.length;
  }
  if (activityPosition) {
    const scroll = el("page-body").querySelector<HTMLElement>(".activity-scroll");
    if (scroll) { scroll.scrollLeft = activityPosition[0]!; scroll.scrollTop = activityPosition[1]!; }
  }
  const bodyBusy = String(state.busy);
  if (el("page-body").getAttribute("aria-busy") !== bodyBusy) el("page-body").setAttribute("aria-busy", bodyBusy);
  if (focusKey && document.activeElement !== focused) for (const button of focusArea!.querySelectorAll<HTMLElement>("button, input")) {
    if ([button.dataset.action, button.dataset.range, button.dataset.page, button.dataset.app, button.dataset.appFile, button.dataset.appSelectAll, button.dataset.process, button.dataset.selectProcess, button.dataset.metric, button.dataset.connectionSort, button.dataset.processSort, button.dataset.processScope, button.dataset.connectionMode, button.dataset.favoritePort, button.dataset.copyPort, button.dataset.portProcess, button.dataset.closePort, button.dataset.portScope, button.dataset.portQuery].includes(focusKey)) { button.focus({ preventScroll: true }); break; }
  }
  const s = state.snapshot;
  if (s) {
    el("nav-cpu").textContent = s.cpu.ready ? percent(s.cpu.usage) : "—";
    el("nav-memory").textContent = !s.memory.error ? percent(s.memory.used / s.memory.total * 100) : "—";
    el("nav-disk").textContent = s.disk.capacityError ? "—" : formatCapacity(s.disk.volume.available);
    el("nav-gpu").textContent = s.gpu.available ? percent(s.gpu.usage) : "—";
    el("nav-gpu").title = s.gpu.available ? t("GPU 使用率") : s.gpu.error || t("不可用");
    const networkReady = s.network.ready && !s.network.error;
    el("nav-network").textContent = networkReady ? `↓ ${formatRate(s.network.download)}` : "—";
    el("nav-network").title = networkReady ? t("下载") : s.network.error || t("采样中");
    const batteryAvailable = !s.battery.error && s.battery.present;
    el("nav-energy").textContent = batteryAvailable ? percent(s.battery.percent) : "—";
    el("nav-energy").title = batteryAvailable ? t("电池电量") : s.battery.error || t("未检测到内置电池");
  }
}
function renderNavigation(selected: Page) {
  for (const button of el("navigation").querySelectorAll<HTMLButtonElement>("button")) {
    if (button.dataset.page === selected) button.setAttribute("aria-current", "page");
    else button.removeAttribute("aria-current");
  }
}
function recordProcessSample() {
  const process = state.processes?.items.find(item => item.pid === state.selectedProcess);
  if (!process || !state.processes) return;
  const at = state.processes.updatedAt;
  const history = (state.processHistory || []).filter(point => Date.parse(point.at) > Date.parse(at) - 60_000);
  if (history.at(-1)?.at !== at) history.push({ at, cpu: process.cpuReady ? process.cpu : null, memory: process.memory, gpu: null, diskRead: null, diskWrite: null, download: null, upload: null, battery: null, power: null });
  state.processHistory = history;
}
async function loadProcesses(live = false) {
  if (state.processes && !state.processError && state.processes.updatedAt === state.snapshot?.processUpdatedAt) return;
  try {
    const result = await readProcesses();
    if (live && state.paused) return;
    state.processes = result; state.processError = state.processes.error; recordProcessSample();
  }
  catch (reason) { state.processError = String(reason); }
}
const refreshProcessView = singleFlight(async () => {
  await loadProcesses(true);
  if (!state.paused && !document.hidden && ["processes", "apps", "connections"].includes(state.page)) render();
});
const refreshConnectionView = singleFlight(async () => {
  await connections(true);
  if (state.page === "connections" && !state.paused && !document.hidden) render();
});
async function readHistory(snapshot: Snapshot, live = false) {
  const historySeconds = state.info!.historySeconds;
  const last = state.history.at(-1);
  const incoming = last ? await System.historySince(last.at) : await System.history();
  if (live && state.paused) return;
  if (!state.snapshot || Date.parse(snapshot.updatedAt) >= Date.parse(state.snapshot.updatedAt)) {
    state.snapshot = snapshot;
    state.history = mergeHistory(state.history, incoming, Date.parse(snapshot.updatedAt), historySeconds);
  }
}
async function receive(snapshot: Snapshot) {
  if (stopped || !state.info) return;
  if (state.snapshot && Date.parse(snapshot.updatedAt) < Date.parse(state.snapshot.updatedAt)) return;
  if (updating || state.paused || document.hidden) {
    if (!queuedSnapshot || Date.parse(snapshot.updatedAt) >= Date.parse(queuedSnapshot.updatedAt)) queuedSnapshot = snapshot;
    return;
  }
  updating = true;
  queuedSnapshot = null;
  const epoch = navigationEpoch;
  try {
    await readHistory(snapshot, true);
    if (state.paused || stopped) return;
    if (document.hidden) { queuedSnapshot ||= snapshot; return; }
    if (errorKind === "data") error("");
    if (epoch === navigationEpoch) render();
    if (["processes", "apps", "connections"].includes(state.page)) void refreshProcessView();
    if (state.page === "connections") void refreshConnectionView();
    if (state.page === "settings" && state.preferences) void refreshTrayPreview();
  } catch (reason) { error(`${t("读取历史失败：")}${String(reason)}${t("。当前显示上次成功的数据。")}`); render(); }
  finally {
    updating = false;
    if (queuedSnapshot && !state.paused && !document.hidden && !stopped) void receive(queuedSnapshot);
  }
}
async function initialize() {
  const result = await System.initialize();
  state.info = result.info;
  state.snapshot = result.snapshot;
  state.history = mergeHistory([], result.history, Date.parse(result.snapshot.updatedAt), result.info.historySeconds);
  el("machine-name").textContent = t("本机");
  el("machine-name").title = result.info.machine.hostname;
  el("machine-detail").textContent = result.info.machine.chip || result.info.machine.architecture;
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
async function connections(live = false, force = false) {
  if (!force && state.connections && !state.connectionError && Date.now() - connectionTime < state.info!.processRefreshSeconds * 1000) return;
  connectionTime = Date.now();
  try {
    const result = await readConnections();
    if (live && state.paused && state.page === "connections") return;
    state.connections = result; state.connectionError = "";
    connectionTime = Date.now();
  } catch (reason) {
    if (!live || !state.paused || state.page !== "connections") state.connectionError = String(reason);
  }
}
function preloadInspection() {
  if (inspectionPreloadScheduled || !state.info || !isMyGo()) return;
  inspectionPreloadScheduled = true;
  const preload = () => {
    if (stopped) return;
    void loadReactPages();
    void loadProcesses().then(() => {
      if (!state.paused && ["processes", "apps", "connections"].includes(state.page)) render();
    });
    void connections(true).then(() => {
      if (!state.paused && state.page === "connections") render();
    });
  };
  if (typeof window.requestIdleCallback === "function") window.requestIdleCallback(preload);
  else requestAnimationFrame(() => { setTimeout(preload, 0); });
}
async function refresh() {
  if (state.busy) return;
  if (!isMyGo()) { error(t("实时数据仅在 RockshinOps 桌面应用中可用。请运行 bun run dev 启动应用。")); return; }
  reactPageError = "";
  state.busy = true; error(""); render();
  try {
    if (!state.info) await initialize();
    else {
      const snapshot = await System.refresh();
      await readHistory(snapshot);
    }
    if (["processes", "apps", "connections"].includes(state.page)) await loadProcesses();
    if (state.page === "connections") await connections(false, true);
  } catch (reason) { error(`${t("连接采集服务失败：")}${String(reason)}${t("。可再次点击刷新重试。")}`); }
  finally { state.busy = false; render(); preloadInspection(); }
}
async function navigate(page: Page, query = "") {
  const current = navigationTarget || state;
  if (page === current.page && query === current.query) return;
  const epoch = ++navigationEpoch;
  navigationTarget = { page, query };
  renderNavigation(page);
  // Paint the selected item before constructing a potentially large table.
  await new Promise<void>(resolve => requestAnimationFrame(() => { setTimeout(resolve, 0); }));
  if (epoch !== navigationEpoch) return;
  navigationTarget = null;
  state.page = page; state.query = query; state.selectedProcess = null;
  if (!metricPages.includes(page)) state.paused = false;
  if (page === "apps" && !["cpu", "memory", "name"].includes(state.sort)) state.sort = "cpu";
  el("content-scroll").scrollTop = 0; render(true);
  if (page === "settings") {
    if (!state.preferences) await loadSettings();
    await loadStartupSettings();
  }
  if (!isMyGo() || !state.info) return;
  if (page === "connections") {
    await Promise.all([loadProcesses(), connections(true), state.portsLoaded ? Promise.resolve() : loadPortFavorites()]);
    if (epoch === navigationEpoch) render();
  } else if (["processes", "apps"].includes(page)) {
    await loadProcesses();
    if (epoch !== navigationEpoch) return;
    render();
  }
}
async function loadPortFavorites() {
  state.portsError = "";
  try { state.favoritePorts = await System.favoritePorts(); state.portsLoaded = true; }
  catch (reason) { state.portsLoaded = false; state.portsError = `${t("读取端口收藏失败：")}${String(reason)}`; }
  render();
}
async function toggleFavoritePort(port: number) {
  if (state.portsSaving || !state.portsLoaded) return;
  state.portsSaving = true; state.portsError = ""; render();
  const favorites = state.favoritePorts || [];
  try { state.favoritePorts = await System.saveFavoritePorts(favorites.includes(port) ? favorites.filter(value => value !== port) : [...favorites, port]); }
  catch (reason) { state.portsError = `${t("保存端口收藏失败：")}${String(reason)}`; }
  finally { state.portsSaving = false; render(); }
}
async function copyPortAddress(address: string) {
  try { await System.copyPortAddress(address); state.portMessage = { text: t("已复制 {{address}}", { address }), error: false }; }
  catch (reason) { state.portMessage = { text: `${t("复制失败：")}${String(reason)}`, error: true }; }
  render();
}
async function inspectPortProcess(pid: number) {
  state.processScope = "all"; state.onlyMine = false; state.processHistory = [];
  await navigate("processes", String(pid));
  if (state.page !== "processes" || state.query !== String(pid)) return;
  state.selectedProcess = pid; recordProcessSample(); render();
}
async function closePortProcess(pid: number) {
  if (state.portAction?.busy || pid <= 1) return;
  const rows = listeningPorts(state.connections?.items || [], state.processes?.items || []);
  const row = rows.find(item => item.pid === pid && item.identity && item.process);
  if (!row) { state.portMessage = { text: t("进程信息不可用，无法结束。"), error: true }; render(); return; }
  const prompt = t("强制退出 {{name}}（PID {{pid}}）以关闭端口 {{port}}？未保存的数据可能丢失。", { name: row.process, pid: row.pid, port: row.port });
  if (!window.confirm(prompt)) return;
  state.portAction = { pid, busy: true }; render();
  try {
    await System.terminateProcess(row.pid, row.identity, true);
    state.portMessage = { text: t("已结束 {{name}}，端口 {{port}} 即将释放。", { name: row.process, port: row.port }), error: false };
    try { await System.refresh(); await loadProcesses(); await connections(true, true); }
    catch (reason) { state.portMessage = { text: `${state.portMessage.text} ${t("刷新失败：")}${String(reason)}`, error: true }; }
  } catch (reason) {
    state.portMessage = { text: `${t("关闭端口失败：")}${String(reason)}`, error: true };
  } finally { state.portAction = undefined; render(); }
}
async function loadSettings() {
  try {
    state.preferences = await System.preferences();
    resetSettingsDrafts(state.preferences); state.preferencesError = "";
  } catch (reason) { state.preferencesError = `${t("无法读取设置：")}${String(reason)}`; }
  if (state.page === "settings") { render(); void refreshTrayPreview(); }
}
let startupRequest = 0;
async function loadStartupSettings() {
  if (state.startupBusy) return;
  const request = ++startupRequest;
  try {
    const startup = await System.startupSettings();
    if (request !== startupRequest) return;
    state.startup = startup; state.startupError = "";
  } catch (reason) {
    if (request !== startupRequest) return;
    state.startupError = `${t("无法读取设置：")}${String(reason)}`;
  }
  if (state.page === "settings") render();
}
async function toggleStartup() {
  if (state.startupBusy || !state.startup?.available) return;
  startupRequest++;
  state.startupBusy = true; state.startupError = ""; render();
  try { state.startup = await System.setOpenAtLogin(!state.startup.enabled); }
  catch (reason) { state.startupError = `${t("无法设置开机自启动：")}${String(reason)}`; }
  finally { state.startupBusy = false; render(); }
}
let trayPreviewRequest = 0;
async function refreshTrayPreview() {
  const request = ++trayPreviewRequest;
  const preview = await System.menuBarPreview([...state.menuBarDraft]).catch(() => null);
  if (request === trayPreviewRequest) { state.trayPreview = preview; if (state.page === "settings") render(); }
}
function resetSettingsDrafts(saved: Preferences) {
  state.hideDockDraft = saved.hideDockOnClose;
  state.updateProxyDraft = saved.updateUseSystemProxy; state.menuBarDraft = [...saved.menuBarItems]; state.languageDraft = saved.language as Language;
}
// Settings save on every change; changes made while a save is in flight are picked up by the loop.
async function saveSettings() {
  if (state.preferencesBusy || !preferencesDirty(state)) return;
  state.preferencesBusy = true; state.preferencesError = "";
  try { while (preferencesDirty(state)) state.preferences = await System.savePreferences({ hideDockOnClose: state.hideDockDraft, menuBarItems: [...state.menuBarDraft], language: state.languageDraft, updateUseSystemProxy: state.updateProxyDraft }); }
  catch (reason) { if (state.preferences) resetSettingsDrafts(state.preferences); state.preferencesError = `${t("保存失败，设置已恢复：")}${String(reason)}${t("。请重试。")}`; }
  finally { state.preferencesBusy = false; render(); void refreshTrayPreview(); }
}
async function terminateSelectedProcess(force: boolean) {
  if (state.processAction?.busy) return;
  const process = state.processes?.items.find(item => item.pid === state.selectedProcess);
  if (!process) return;
  const prompt = force
    ? t("强制退出 {{name}}（PID {{pid}}）？未保存的数据可能丢失。", { name: process.name, pid: process.pid })
    : t("请求退出 {{name}}（PID {{pid}}）？进程可能拒绝或延迟退出。", { name: process.name, pid: process.pid });
  if (!window.confirm(prompt)) return;
  const action = { pid: process.pid, busy: true, error: false, message: t("正在发送退出请求…") };
  state.processAction = action; render();
  try {
    await System.terminateProcess(process.pid, process.identity, force);
    action.message = force ? t("强制退出请求已发送，等待进程列表更新。") : t("退出请求已发送；若进程仍在运行，可稍后强制退出。");
    try { await System.refresh(); await loadProcesses(); }
    catch (reason) { action.message += ` ${t("刷新进程列表失败：")}${String(reason)}`; }
  } catch (reason) {
    action.error = true; action.message = `${t("退出失败：")}${String(reason)}`;
  } finally { action.busy = false; render(); }
}
async function hide() {
  try { await System.hideWindow(); }
  catch (reason) { error(`${t("无法收起窗口：")}${String(reason)}`, "action"); }
}
document.addEventListener("click", event => {
  const processRow = (event.target as Element).closest<HTMLElement>("[data-select-process]");
  if (processRow) {
    const pid = Number(processRow.dataset.selectProcess);
    if (state.selectedProcess !== pid) state.processHistory = [];
    state.selectedProcess = pid; recordProcessSample(); render(); return;
  }
  const button = (event.target as Element).closest<HTMLButtonElement>("button");
  if (!button || button.disabled) return;
  if (button.dataset.reviewApp) { void reviewApp(button.dataset.reviewApp); return; }
  if (button.dataset.action === "toggle-startup") { void toggleStartup(); return; }
  if (button.dataset.action === "load-startup") { void loadStartupSettings(); return; }
  if (button.dataset.action === "toggle-hide-dock") { state.hideDockDraft = !state.hideDockDraft; render(); void saveSettings(); return; }
  if (button.dataset.action === "toggle-update-proxy") { state.updateProxyDraft = !state.updateProxyDraft; render(); void saveSettings(); return; }
  if (button.dataset.metric) {
    const selected = new Set(state.menuBarDraft);
    if (selected.has(button.dataset.metric)) selected.delete(button.dataset.metric); else selected.add(button.dataset.metric);
    state.menuBarDraft = menuBarChoices.map(([id]) => id).filter(id => selected.has(id)); render(); void saveSettings(); return;
  }
  if (button.dataset.page) { void navigate(button.dataset.page as Page); return; }
  if (button.dataset.app) { void navigate("processes", button.dataset.app); return; }
  if (button.dataset.process) { void navigate("processes", button.dataset.process); return; }
  if (button.dataset.processScope) { state.processScope = button.dataset.processScope as ViewState["processScope"]; state.onlyMine = false; render(true); return; }
  if (button.dataset.processSort) { const key = button.dataset.processSort as NonNullable<ViewState["processOrder"]>["key"]; state.processOrder = { key, descending: state.processOrder?.key === key ? !state.processOrder.descending : key !== "name" }; render(); return; }
  if (button.dataset.connectionMode) { state.connectionMode = button.dataset.connectionMode as "all" | "ports"; state.query = ""; state.portMessage = undefined; state.portAction = undefined; render(true); return; }
  if (button.dataset.favoritePort) { void toggleFavoritePort(Number(button.dataset.favoritePort)); return; }
  if (button.dataset.copyPort) { void copyPortAddress(button.dataset.copyPort); return; }
  if (button.dataset.portProcess) { void inspectPortProcess(Number(button.dataset.portProcess)); return; }
  if (button.dataset.closePort) { void closePortProcess(Number(button.dataset.closePort)); return; }
  if (button.dataset.portScope) { state.portScope = button.dataset.portScope as ViewState["portScope"]; render(); return; }
  if (button.dataset.portQuery) { state.query = `:${button.dataset.portQuery}`; state.portScope = "all"; render(true); return; }
  if (button.dataset.connectionSort) { state.connectionSort = nextConnectionSort(state.connectionSort, button.dataset.connectionSort as ConnectionSortKey); render(); return; }
  if (button.dataset.range) { state.range = Number(button.dataset.range); render(); return; }
  switch (button.dataset.action) {
    case "load-port-favorites": void loadPortFavorites(); break;
    case "clear-port-filters": state.query = ""; state.portScope = "all"; render(true); break;
    case "quit-process": void terminateSelectedProcess(false); break;
    case "force-quit-process": void terminateSelectedProcess(true); break;
    case "close-process": state.selectedProcess = null; render(); break;
    case "pause": state.paused = !state.paused; render(); if (!state.paused) void refresh(); break;
    case "refresh": void refresh(); break;
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
el("page-body").addEventListener("scroll", event => {
  const scroll = event.target as HTMLElement;
  if (!scroll.classList.contains("activity-scroll")) return;
  const next = tableWindow(scroll.scrollTop, scroll.clientHeight);
  if (next.start !== state.tableWindow?.start || next.count !== state.tableWindow?.count) render();
}, { capture: true, passive: true });
new ResizeObserver(() => {
  if (state.page === "processes" || state.page === "connections") render();
}).observe(el("page-body"));
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
document.querySelector(".sidebar")!.setAttribute("aria-label", t("RockshinOps 导航"));
el("navigation").setAttribute("aria-label", t("监控页面"));
el("machine-detail").textContent = t("正在连接本机…");
render(true);
if (isMyGo()) {
  const offLocale = onLocaleChange(language => {
    void i18n.changeLanguage(language);
    document.documentElement.lang = language;
    el("navigation").innerHTML = navigation();
    renderNavigation(navigationTarget?.page ?? state.page);
    el("machine-name").textContent = t("本机");
    document.querySelector(".sidebar")!.setAttribute("aria-label", t("RockshinOps 导航"));
    el("navigation").setAttribute("aria-label", t("监控页面"));
    state.info = null; state.processes = null; state.connections = null;
    inspectionPreloadScheduled = false;
    state.processError = ""; state.connectionError = "";
    error(""); render(true); void refresh();
  });
  document.addEventListener("visibilitychange", () => {
    if (!document.hidden && queuedSnapshot && !state.paused) void receive(queuedSnapshot);
    if (!document.hidden && state.page === "settings") void loadStartupSettings();
  });
  window.addEventListener("focus", () => {
    if (state.page === "settings") void loadStartupSettings();
  });
  const off = events.metricsChanged.on(snapshot => { void receive(snapshot); });
  const offSettings = events.settingsOpened.on(() => { void navigate("settings"); });
  const offMenuBarSettings = events.menuBarSettingsOpened.on(() => { void navigate("menubar"); });
  window.addEventListener("pagehide", () => { stopped = true; off(); offSettings(); offMenuBarSettings(); offLocale(); }, { once: true });
  void refresh();
} else {
  el("machine-detail").textContent = t("浏览器预览");
  error(t("实时数据仅在 RockshinOps 桌面应用中可用。请运行 bun run dev 启动应用。"));
}
