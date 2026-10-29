import { windowRows, type TableWindow } from "./table-window";
import { trendTime } from "./history";
import { t, i18n } from "./i18n";
import type { AppInfo, Snapshot, Trend, ProcessList, Connections, ScanResult, AppUsage, Process, InstalledApp, AppReview } from "./mygo";
import { capacityParts, duration, escapeHTML as e, formatBytes, formatCapacity, formatMemory, formatRate, memoryParts, percent, rateParts, timeLabel } from "./format";
import { dots, lineChart, type TrendKey } from "./charts";
import { icon } from "./icons";
import { sortControl } from "./dropdown";
import { type SettingsState } from "./settings";
import { connectionRows, splitEndpoint, type ConnectionSort, type ConnectionSortKey } from "./connections";

export const pages = [
  { id: "overview", title: "概览", group: "" },
  { id: "processes", title: "进程", group: "" },
  { id: "connections", title: "连接", group: "" },
  { id: "cpu", title: "CPU", group: "硬件" },
  { id: "memory", title: "内存", group: "硬件" },
  { id: "gpu", title: "GPU", group: "硬件" },
  { id: "disk", title: "磁盘", group: "硬件" },
  { id: "network", title: "网络", group: "硬件" },
  { id: "energy", title: "能耗", group: "硬件" },
  { id: "apps", title: "App", group: "系统" },
  { id: "cleanup", title: "清理", group: "系统" },
  { id: "menubar", title: "菜单栏管理", group: "系统" },
  { id: "about", title: "关于本机", group: "系统" },
  { id: "settings", title: "设置", group: "系统" },
] as const;
export type Page = (typeof pages)[number]["id"];
export const metricPages: Page[] = ["overview", "cpu", "memory", "gpu", "disk", "network", "energy"];

export interface ViewState extends SettingsState {
  page: Page;
  tableWindow?: TableWindow;
  info: AppInfo | null;
  snapshot: Snapshot | null;
  history: Trend[];
  range: number;
  paused: boolean;
  processes: ProcessList | null;
  connections: Connections | null;
  connectionSort: ConnectionSort | null;
  selectedProcess: number | null;
  processHistory?: Trend[];
  processAction?: { pid: number; busy: boolean; error: boolean; message: string };
  processScope?: "all" | "apps" | "mine" | "system";
  processOrder?: { key: "name" | "cpu" | "memory" | "threads" | "pid" | "userId"; descending: boolean };
  scan: ScanResult | null;
  query: string;
  sort: string;
  onlyMine: boolean;
  busy: boolean;
  processError: string;
  connectionError: string;
  scanError: string;
  scanning: boolean;
  cleanupBusy: boolean;
  cleanupMessage: string;
  spaceMode: boolean;
  spaceHistory: string[];
  appMode: boolean;
  installedApps: InstalledApp[] | null;
  appReview: AppReview | null;
  appSelection: string[];
  appBusy: boolean;
  appSearching: boolean;
  appMessage: string;
}

function isMac(state: ViewState): boolean { return state.info?.machine.platform === "darwin"; }
function appIcon(path: string, native: boolean): string { if (!native) return `<span class="app-icon app-letter">${icon("apps")}</span>`; return `<img class="app-icon" src="appicon://localhost/?path=${encodeURIComponent(path)}" alt="">`; }
function unavailable(message = t("暂不可用")): string { return `<span class="unavailable">${e(message)}</span>`; }
function stat(label: string, value: string): string { return `<div><dt>${e(label)}</dt><dd>${value}</dd></div>`; }
function stats(items: [string, string][]): string { return `<dl class="stats-row">${items.map(([k, v]) => stat(k, v)).join("")}</dl>`; }
function badge(label: string, kind = ""): string { return `<span class="badge ${kind}">${e(label)}</span>`; }
function empty(title: string, detail: string, action = ""): string { return `<div class="empty-state"><h2>${e(title)}</h2><p>${e(detail)}</p>${action}</div>`; }
function note(text: string): string { return `<p class="source-note">${e(text)}</p>`; }
function notice(text: string): string { return `<div class="notice">${e(text)}</div>`; }
function value(parts: { value: string; unit: string } | null, label = t("采样中")): string { return parts ? `<span>${e(parts.value)}</span><small>${e(parts.unit)}</small>` : `<span class="waiting-value">${e(label)}</span>`; }
function metricValue(number: number | null, unit: string, digits = 0): string { return value(number == null ? null : { value: number.toFixed(digits), unit }); }
function chart(state: ViewState, key: TrendKey, title: string, max = 100, secondary?: TrendKey): string {
  return lineChart(state.history, key, new Date(state.snapshot!.updatedAt).getTime(), state.range, title, max, secondary, state.info?.refreshSeconds || 2);
}

export function navigation(): string {
  let previous = "";
  return pages.map(page => {
    const heading = page.group && page.group !== previous ? `<p class="nav-group">${t(page.group)}</p>` : "";
    previous = page.group;
    return `${heading}<button type="button" class="nav-item" data-page="${page.id}" ${page.id === "overview" ? 'aria-current="page"' : ""}><span class="nav-icon">${icon(page.id === "energy" ? "battery" : page.id)}</span><span>${t(page.title)}</span><small id="nav-${page.id}"></small></button>`;
  }).join("");
}

export function actionControls(state: ViewState): string {
  if (state.page === "menubar") return "";
  if (state.page === "settings") return `<span class="settings-shortcut">${isMac(state) ? "⌘," : "Ctrl+,"}</span>`;
  const ranges = [[60, t("1 分钟")], [300, t("5 分钟")], [900, t("15 分钟")], [3600, t("1 小时")]] as const;
  const history = metricPages.includes(state.page) ? `<div class="segmented" role="group" aria-label="${t("历史时间范围")}">${ranges.map(([seconds, label]) => `<button data-range="${seconds}" class="${state.range === seconds ? "selected" : ""}" aria-pressed="${state.range === seconds}">${label}</button>`).join("")}</div>` : "";
  const pause = metricPages.includes(state.page) || ["processes", "connections"].includes(state.page) ? `<button class="icon-button pause-button ${state.paused ? "selected" : ""}" data-action="pause" aria-pressed="${state.paused}" aria-label="${state.paused ? t("恢复实时视图") : t("暂停实时视图")}" title="${state.paused ? t("恢复实时视图") : t("暂停视图，菜单栏继续采集")}">${icon(state.paused ? "play" : "pause")}</button>` : "";
  if (["processes", "connections"].includes(state.page)) return pause;
  return `${history}${pause}<button class="icon-button" data-action="refresh" ${state.busy ? 'aria-disabled="true" aria-busy="true"' : ""} aria-label="${state.page === "cleanup" ? t("重新扫描") : t("立即刷新")}" title="${state.page === "cleanup" ? t("重新扫描") : t("立即刷新")}">${icon("refresh")}</button>`;
}

export function filterControls(state: ViewState): string {
  if (!["processes", "connections", "apps"].includes(state.page) || (state.page === "apps" && !!state.appReview)) return "";
  const sorting = ["processes", "connections"].includes(state.page) || (state.page === "apps" && state.appMode) ? "" : sortControl(state.sort, state.page === "processes");
  return `<label class="search-field">${icon("search")}<input id="search" type="search" placeholder="${["processes", "connections"].includes(state.page) ? t("搜索") : state.page === "apps" && state.appMode ? t("搜索 App 名称或路径") : t("搜索名称或 PID")}" aria-label="${t("搜索")}${t(pages.find(p => p.id === state.page)!.title)}" value="${e(state.query)}" autocomplete="off" spellcheck="false"/></label>${sorting}${state.page === "processes" ? `<div class="segmented process-scopes" role="group" aria-label="${t("进程范围")}">${([["all", "全部"], ["apps", "App"], ["mine", "我的进程"], ["system", "系统"]] as const).map(([key, label]) => `<button data-process-scope="${key}" aria-pressed="${(state.processScope || "all") === key}" class="${(state.processScope || "all") === key ? "selected" : ""}">${t(label)}</button>`).join("")}</div>` : ""}`;
}

function card(state: ViewState, page: Page, color: string, title: string, label: string, main: string, items: [string, string][], keys: TrendKey[], max = 100, status = ""): string {
  return `<button class="metric-card ${color}" data-page="${page}" type="button"><div class="card-heading"><span class="card-icon content-icon">${icon(page === "energy" ? "battery" : page)}</span><h2>${e(title)}</h2><span class="card-chevron">${icon("chevron")}</span></div><p class="metric-label">${e(label)}</p><div class="metric-value">${main}${status}</div>${stats(items)}${dots(state.history, keys, new Date(state.snapshot!.updatedAt).getTime(), state.range, title + t("历史"), max)}</button>`;
}

function overview(state: ViewState): string {
  const s = state.snapshot!, m = s.memory, b = s.battery;
  const memoryOK = !m.error && m.total > 0;
  const batteryOK = b.present && !b.error;
  const batteryState = b.error ? t("电池不可用") : !b.present ? t("未检测到电池") : b.charging ? t("正在充电") : b.externalPower ? t("已连接电源") : t("使用电池");
  const cards = [
    card(state, "cpu", "cpu", "CPU", s.cpu.error || t("当前"), metricValue(s.cpu.ready ? s.cpu.usage : null, "%"), [[t("用户"), s.cpu.ready ? percent(s.cpu.user) : "—"], [t("系统"), s.cpu.ready ? percent(s.cpu.system) : "—"], [t("逻辑核心"), String(state.info?.machine.logicalCPUs || "—")]], ["cpu"]),
    card(state, "memory", "memory", t("内存"), memoryOK ? `${t("已使用，共")} ${formatMemory(m.total)}` : m.error, value(memoryOK ? memoryParts(m.used) : null, t("不可用")), isMac(state) ? [[t("App 内存"), memoryOK ? formatMemory(m.app) : "—"], [t("联动内存"), memoryOK ? formatMemory(m.wired) : "—"], [t("已压缩"), memoryOK ? formatMemory(m.compressed) : "—"]] : [[t("可用"), memoryOK ? formatMemory(m.available) : "—"], [t("已使用交换空间"), m.swapError ? "—" : formatMemory(m.swapUsed)], [t("交换空间总量"), m.swapError ? "—" : formatMemory(m.swapTotal)]], ["memory"], 100, memoryOK && m.pressure !== "unavailable" ? badge(t(m.pressure), m.pressure === "elevated" || m.pressure === "critical" ? "warning" : "") : ""),
    card(state, "gpu", "gpu", "GPU", s.gpu.name || t("GPU 性能"), metricValue(s.gpu.available ? s.gpu.usage : null, "%"), [[t("内存"), s.gpu.available ? formatMemory(s.gpu.memory) : "—"], [t("范围平均"), trendAverage(state, "gpu")], [t("范围峰值"), trendPeak(state, "gpu")]], ["gpu"]),
    card(state, "disk", "disk", t("磁盘"), s.disk.capacityError || `${t("可用，共")} ${formatCapacity(s.disk.volume.total)}`, value(!s.disk.capacityError ? capacityParts(s.disk.volume.available) : null, t("不可用")), [[t("读取速度"), s.disk.ready ? formatRate(s.disk.readRate) : t("采样中")], [t("写入速度"), s.disk.ready ? formatRate(s.disk.writeRate) : t("采样中")], [t("驱动累计写入"), s.disk.error ? "—" : formatCapacity(s.disk.writeBytes)]], ["diskRead", "diskWrite"], 0),
    card(state, "network", "network", t("网络"), t("下载速度 · 物理网络接口"), value(s.network.ready ? rateParts(s.network.download) : null), [[t("上传速度"), s.network.ready ? formatRate(s.network.upload) : t("采样中")], [t("已接收"), s.network.error ? "—" : formatCapacity(s.network.received)], [t("已发送"), s.network.error ? "—" : formatCapacity(s.network.sent)]], ["download", "upload"], 0),
    card(state, "energy", "memory", t("电池"), batteryState, metricValue(batteryOK ? b.percent : null, "%"), [[b.charging ? t("充满还需") : t("预计剩余"), batteryOK && b.minutes > 0 ? duration(b.minutes * 60) : t("估算中")], [t("系统负载"), b.systemWatts != null ? `${b.systemWatts.toFixed(1)} W` : "—"], [t("健康度"), b.health != null ? percent(b.health) : "—"]], ["battery"]),
  ];
  return `<div class="overview-grid">${cards.join("")}${memoryComposition(state)}${appSummary(state, s.topMemory)}${cpuSummary(s.topCPU)}</div>`;
}

function trendNumbers(state: ViewState, key: TrendKey): number[] {
  const cutoff = new Date(state.snapshot!.updatedAt).getTime() - state.range * 1000;
  return state.history.filter(p => trendTime(p) >= cutoff && p[key] != null).map(p => p[key]!);
}
function trendAverage(state: ViewState, key: TrendKey): string { const v = trendNumbers(state, key); return v.length ? percent(v.reduce((a, b) => a + b, 0) / v.length) : "—"; }
function trendPeak(state: ViewState, key: TrendKey): string { const v = trendNumbers(state, key); return v.length ? percent(Math.max(...v)) : "—"; }

function memoryComposition(state: ViewState): string {
  const m = state.snapshot!.memory;
  const items: [string, number, string][] = isMac(state) ? [[t("App 内存"), m.app, "1"], [t("联动内存"), m.wired, "2"], [t("已压缩"), m.compressed, "3"], [t("可用"), m.available, "0"]] : [[t("已使用"), m.used, "1"], [t("可用"), m.available, "0"]];
  const segments: string[] = [];
  let offset = 0;
  for (const [, bytes, color] of items) {
    const portion = m.total ? bytes / m.total * 100 : 0;
    segments.push(`<circle class="ring-${color}" cx="60" cy="60" r="48" pathLength="100" stroke-dasharray="${portion} ${100 - portion}" stroke-dashoffset="${-offset}"/>`);
    offset += portion;
  }
  return `<section class="summary-card memory"><h2><span class="content-icon">${icon("memory")}</span>${t("内存构成")}</h2>${m.error ? unavailable(m.error) : `<div class="memory-composition"><div class="memory-ring"><svg viewBox="0 0 120 120" aria-hidden="true">${segments.join("")}</svg><div><strong>${percent(m.used / m.total * 100)}</strong><span>${t("已使用")}</span></div></div><dl class="ring-legend">${items.map(([name, bytes, color]) => `<div><dt><i class="legend-dot ring-${color}"></i>${name}</dt><dd>${formatMemory(bytes)}</dd></div>`).join("")}</dl></div>`}${note(t("可用空间包含可回收部分；缓存单独列于内存详情。"))}</section>`;
}

function appSummary(state: ViewState, apps: AppUsage[]): string {
  const max = Math.max(1, ...apps.map(app => app.memory));
  return `<section class="summary-card memory"><h2><span class="content-icon">${icon("apps")}</span>${t("App 内存占用")}</h2><div class="rank-list">${apps.slice(0, 5).map(app => `<button class="rank-item" data-app="${e(app.path)}">${appIcon(app.path, isMac(state))}<span class="rank-copy"><span class="rank-heading"><strong>${e(app.name)}</strong><span>${formatMemory(app.memory)}</span></span><span class="rank-track"><i style="transform:scaleX(${app.memory / max})"></i></span></span></button>`).join("") || unavailable(t("等待进程采样"))}</div>${note((isMac(state) ? t("按 App 包归并进程的物理内存占用。") : t("按可执行文件路径归并进程的常驻内存占用。")))}</section>`;
}

function cpuSummary(processes: Process[]): string {
  const max = Math.max(1, ...processes.map(p => p.cpu));
  return `<section class="summary-card cpu"><h2><span class="content-icon">${icon("cpu")}</span>${t("CPU 活跃进程")}</h2><div class="rank-list">${processes.slice(0, 5).map(p => `<button class="rank-item" data-process="${p.pid}"><span class="app-letter">${e(p.name.slice(0, 1).toUpperCase())}</span><span class="rank-copy"><span class="rank-heading"><strong>${e(p.name)}</strong><span>${p.cpuReady ? percent(p.cpu) : t("采样中")}</span></span><span class="rank-track"><i style="transform:scaleX(${p.cpu / max})"></i></span></span></button>`).join("") || unavailable(t("等待进程采样"))}</div>${note(t("单核占满为 100%；这不是 App 功耗排名。"))}</section>`;
}

function panel(title: string, content: string, color = "neutral", extra = ""): string { return `<section class="detail-card ${color} ${extra}"><h2>${e(title)}</h2>${content}</section>`; }
function facts(items: [string, string][]): string { return `<dl class="fact-list">${items.map(([label, content]) => `<div><dt>${e(label)}</dt><dd>${content}</dd></div>`).join("")}</dl>`; }

function cpuView(state: ViewState): string {
  const s = state.snapshot!, c = s.cpu;
  return `${c.error ? notice(c.error) : ""}<div class="detail-grid">${panel(t("处理器使用率"), `<div class="detail-value">${metricValue(c.ready ? c.usage : null, "%")}</div>${stats([[t("用户"), c.ready ? percent(c.user) : "—"], [t("系统"), c.ready ? percent(c.system) : "—"], [t("空闲"), c.ready ? percent(100 - c.usage) : "—"]])}${chart(state, "cpu", t("CPU 使用率"))}`, "cpu", "span-two")}${panel(t("处理器"), facts([[t("芯片"), e(state.info?.machine.chip || "—")], [t("物理核心"), String(state.info?.machine.physicalCPUs || "—")], [t("逻辑核心"), String(state.info?.machine.logicalCPUs || "—")], [t("平均使用率"), trendAverage(state, "cpu")], [t("峰值使用率"), trendPeak(state, "cpu")], [t("温度"), unavailable(t("未接入"))]]) + note(t("CPU 温度尚未接入可靠的无提权接口；不使用电池温度替代。")), "cpu")}${panel(t("各逻辑核心"), `<div class="core-grid">${c.ready ? c.cores.map((v, i) => `<div class="core-item"><span>${t("核心")} ${i + 1}</span><strong>${percent(v)}</strong><div class="core-track"><i style="transform:scaleX(${v / 100})"></i></div></div>`).join("") : unavailable(t("等待第二个 CPU 样本"))}</div>`, "cpu", "span-two")}${cpuSummary(s.topCPU)}</div>${note((isMac(state) ? t("来源：Mach host_processor_info。总使用率按所有逻辑核心的时钟增量加权计算。") : t("来源：操作系统逐核 CPU 时间计数。总使用率按所有逻辑核心的时间增量加权计算；Linux I/O 等待计为空闲。")))}`;
}

function memoryView(state: ViewState): string {
  const m = state.snapshot!.memory;
  if (m.error) return empty(t("内存暂不可用"), m.error);
  return `<div class="detail-grid">${panel(t("内存使用"), `<div class="detail-value">${value(memoryParts(m.used))}<span class="value-total">/ ${formatMemory(m.total)}</span></div>${m.pressure === "unavailable" ? "" : badge(`${t("内存压力：")}${t(m.pressure)}`, m.pressure === "normal" ? "" : "warning")}${chart(state, "memory", t("物理内存使用率"))}`, "memory", "span-two")}${memoryComposition(state)}${panel(t("内存详情"), facts([...(isMac(state) ? [[t("App 内存"), formatMemory(m.app)], [t("联动内存"), formatMemory(m.wired)], [t("压缩占用"), formatMemory(m.compressed)]] as [string, string][] : []), ...(state.info?.machine.platform !== "windows" ? [[t("文件缓存与可清除页"), formatMemory(m.cached)]] as [string, string][] : []), [t("可用"), formatMemory(m.available)], [t("已使用交换空间"), m.swapError ? unavailable(m.swapError) : formatMemory(m.swapUsed)], [t("交换空间总量"), m.swapError ? "—" : formatMemory(m.swapTotal)]]), "memory", "span-two")}${appSummary(state, state.snapshot!.topMemory)}</div>${note((isMac(state) ? t("来源：Mach host_statistics64 与 sysctl。已用 = App + 联动 + 压缩；App = 活跃 + 非活跃 + 推测页 − 可清除 − 文件页。内存使用二进制 GiB/MiB，磁盘容量使用十进制 GB/MB。") : t("已用 = 总物理内存 − 系统报告的可用内存。Linux 包含可回收缓存的可用估计，Windows 使用可用物理页；交换空间在 Windows 表示页面文件。内存使用 GiB/MiB。")))}`;
}

function gpuView(state: ViewState): string {
  const g = state.snapshot!.gpu;
  if (!g.available) return empty(t("GPU 性能暂不可用"), g.error);
  return `<div class="detail-grid">${panel(g.name || "GPU", `<div class="detail-value">${metricValue(g.usage, "%")}</div>${stats([[t("已用内存"), formatMemory(g.memory)], [t("范围平均"), trendAverage(state, "gpu")], [t("范围峰值"), trendPeak(state, "gpu")]])}${chart(state, "gpu", t("GPU 使用率"))}`, "gpu", "span-two")}${panel(t("采集信息"), facts([[t("设备"), e(g.name)], [t("来源"), isMac(state) ? "IOAccelerator" : "AMD sysfs / NVIDIA SMI"], [t("计数"), isMac(state) ? "Device Utilization %" : "GPU utilization %"], [t("内存"), isMac(state) ? "In use system memory" : "VRAM used"]]) + note((isMac(state) ? t("显示首个提供完整计数的 GPU。硬件与 macOS 版本变化可能导致此接口不可用。") : t("Linux 优先显示首个提供计数的 AMD GPU，否则查询 NVIDIA 的首个 GPU。Windows 查询 NVIDIA 首个 GPU。驱动未提供的计数显示不可用。"))), "gpu")}</div>`;
}

function diskView(state: ViewState): string {
  const d = state.snapshot!.disk;
  const portion = d.volume.total ? (1 - d.volume.available / d.volume.total) : 0;
  return `${d.capacityError ? notice(d.capacityError) : ""}${d.error ? notice(d.error) : ""}<div class="detail-grid">${panel(t("剩余可写空间"), `<div class="detail-value">${value(!d.capacityError ? capacityParts(d.volume.available) : null, t("不可用"))}</div><p class="muted">${t("共")} ${formatCapacity(d.volume.total)}</p><div class="capacity-track"><i style="transform:scaleX(${portion})"></i></div><div class="capacity-legend"><span>${t("不可用")} ${formatCapacity(d.volume.total - d.volume.available)}</span><span>${t("剩余")} ${(100 * (1 - portion)).toFixed(1)}%</span></div>${facts([[t("文件系统"), e(d.volume.format)], [t("挂载位置"), `<span class="path">${e(d.volume.mount)}</span>`]])}`, "disk", "span-two")}${panel(t("常驻菜单栏"), `<div class="tray-preview">${icon("disk")}<strong>${d.capacityError ? "—" : formatCapacity(d.volume.available)}</strong></div><p class="body-copy">${t("关闭窗口后，容量仍会在菜单栏持续更新。")}</p><button class="secondary-button" data-action="hide">${icon("hide")}${t("收起到菜单栏")}</button>${note(`${t("每")} ${state.info?.diskRefreshSeconds || 30} ${t("秒采集容量。可写空间不含系统可能回收的空间。")}`)}`, "disk")}${panel(t("磁盘吞吐"), stats([[t("读取"), d.ready ? formatRate(d.readRate) : t("采样中")], [t("写入"), d.ready ? formatRate(d.writeRate) : t("采样中")], [t("累计写入"), formatCapacity(d.writeBytes)]]) + chart(state, "diskRead", t("磁盘读取与写入"), 0, "diskWrite") + `<div class="chart-legend"><span>${t("读取")}</span><span>${t("写入")}</span></div>`, "disk", "span-two")}${panel(t("统计口径"), facts([[t("容量"), t("用户主目录所在卷")], [t("I/O 范围"), (isMac(state) ? t("所有块存储驱动") : t("系统磁盘计数"))], [t("累计读取"), formatCapacity(d.readBytes)], [t("累计写入"), formatCapacity(d.writeBytes)]]) + note((isMac(state) ? t("I/O 包含外置存储和系统报告的块设备；累计计数从驱动启动计起。APFS 容量只读数据卷一次。") : t("Windows 汇总固定磁盘卷的 I/O；Linux 汇总硬件块设备，排除分区和虚拟叠加设备。容量只读取用户主目录所在文件系统，不叠加各卷。"))), "disk")}</div>`;
}

function networkView(state: ViewState): string {
  const n = state.snapshot!.network;
  return `${n.error ? notice(n.error) : ""}<div class="detail-grid">${panel(t("网络速度"), `<div class="dual-value"><div><p>${t("下载")}</p><div class="detail-value">${value(n.ready ? rateParts(n.download) : null)}</div></div><div><p>${t("上传")}</p><div class="detail-value">${value(n.ready ? rateParts(n.upload) : null)}</div></div></div>${chart(state, "download", t("网络下载与上传"), 0, "upload")}<div class="chart-legend"><span>${t("下载")}</span><span>${t("上传")}</span></div>`, "network", "span-two")}${panel(t("累计流量"), facts([[t("已接收"), formatCapacity(n.received)], [t("已发送"), formatCapacity(n.sent)], [t("汇总接口"), (isMac(state) ? t("en* 物理接口") : t("系统识别的硬件接口"))]]) + note(t("VPN、桥接与虚拟接口单独列出，避免重复累计同一份流量。")), "network")}</div><section class="table-section"><h2>${t("网络接口")}</h2><table><thead><tr><th>${t("接口")}</th><th>${t("状态")}</th><th>${t("下载")}</th><th>${t("上传")}</th><th>${t("已接收")}</th><th>${t("已发送")}</th></tr></thead><tbody>${n.interfaces.map(item => `<tr><td><strong>${e(item.name)}</strong>${item.included ? `<span class="inline-note">${t("计入汇总")}</span>` : ""}</td><td>${item.up ? badge(t("启用")) : `<span class="muted">${t("未启用")}</span>`}</td><td>${item.ready ? formatRate(item.download) : "—"}</td><td>${item.ready ? formatRate(item.upload) : "—"}</td><td>${formatCapacity(item.received)}</td><td>${formatCapacity(item.sent)}</td></tr>`).join("")}</tbody></table></section>${note((isMac(state) ? t("来源：路由接口表 NET_RT_IFLIST2 的 64 位计数。新出现或重置的接口先建立基线，随后显示速率。") : t("Windows 使用 GetIfEntry2，Linux 使用 /proc/net/dev 与 sysfs 设备标识。汇总硬件接口，所有接口单独列出；新接口或计数重置后重新建立基线。")))}`;
}

function energyView(state: ViewState): string {
  const b = state.snapshot!.battery;
  if (b.error) return empty(t("电池暂不可用"), b.error);
  const details = !b.present ? empty(t("未检测到内置电池"), t("当前设备未报告内置电池；进程 CPU 活动仍可查看。")) : `<div class="detail-grid">${panel(t("电池"), `<div class="detail-value">${metricValue(b.percent, "%")}</div>${badge(b.charging ? t("正在充电") : b.externalPower ? t("已连接电源") : t("使用电池"))}${stats([[b.charging ? t("充满还需") : t("预计剩余"), b.minutes > 0 ? duration(b.minutes * 60) : t("估算中")], [t("健康度"), b.health != null ? percent(b.health) : "—"], [t("循环次数"), b.cycles == null ? "—" : String(b.cycles)]])}${chart(state, "battery", t("电池电量"))}`, "memory", "span-two")}${panel(t("电源信息"), facts([[t("供电状态"), b.externalPower ? t("外部电源") : t("电池")], [t("电池充放电功率"), b.watts != null ? `${Math.abs(b.watts).toFixed(1)} W${b.watts >= 0 ? t("（充入）") : t("（输出）")}` : "—"], [t("电池温度"), b.temperature != null ? `${b.temperature.toFixed(1)} °C` : "—"], [t("系统负载"), b.systemWatts != null ? `${b.systemWatts.toFixed(1)} W` : unavailable()]]) + note((isMac(state) ? t("电池功率按电压 × 瞬时电流计算。温度是电池温度，不是 CPU 温度。") : t("Windows 显示系统电池状态；Linux 显示 sysfs 中首个系统电池，并读取驱动提供的功率、健康度与温度。温度是电池温度。"))), "energy")}${panel(t("系统负载功率"), b.systemWatts != null ? `<div class="detail-value">${metricValue(b.systemWatts, "W", 1)}</div>${chart(state, "power", t("系统负载功率"), 0)}${note(t("来自 AppleSmartBattery 的 PowerTelemetryData.SystemLoad 硬件遥测，非插座测量值。"))}` : unavailable(t("当前硬件未提供系统功率遥测")), "energy", "span-two")}${cpuSummary(state.snapshot!.topCPU)}</div>`;
  return details;
}

// Each section retains only its latest immutable IPC inputs and markup.
// Snapshot-only footer updates do not rebuild an unchanged inspection table.
function cachedMarkup(render: (state: ViewState) => string, inputs: (state: ViewState) => unknown[]) {
  let previous: unknown[] = [], html = "";
  return (state: ViewState): string => {
    const next = inputs(state);
    if (next.length !== previous.length || next.some((value, i) => value !== previous[i])) {
      html = render(state); previous = next;
    }
    return html;
  };
}

function orderedProcesses(state: ViewState): Process[] {
  const query = state.query.toLowerCase();
  const result = (state.processes?.items || []).filter(p => (state.processScope === "apps" ? !!p.app : state.processScope === "mine" ? p.own : state.processScope === "system" ? !p.own : !state.onlyMine || p.own) && (!query || `${p.name} ${p.pid} ${p.path}`.toLowerCase().includes(query)));
  if (!state.processOrder) result.sort((a, b) => state.sort === "name" ? a.name.localeCompare(b.name) : (state.sort === "memory" ? b.memory - a.memory : state.sort === "readRate" ? b.readRate - a.readRate : state.sort === "writeRate" ? b.writeRate - a.writeRate : b.cpu - a.cpu) || a.pid - b.pid);
  if (state.processOrder) {
    const { key, descending } = state.processOrder;
    result.sort((a, b) => (key === "name" ? a.name.localeCompare(b.name) : Number(a[key]) - Number(b[key])) * (descending ? -1 : 1) || a.pid - b.pid);
  }
  return result;
}

function processIcon(process: Process, state: ViewState): string {
  return process.app ? appIcon(process.app, isMac(state)) : `<span class="app-icon process-system-icon">${icon("settings")}</span>`;
}

export function processSubtitle(state: ViewState): string {
  if (!state.processes) return t("正在读取进程");
  return `${state.processes.items.length} ${t("个进程")} · ${state.processes.items.reduce((sum, process) => sum + process.threads, 0).toLocaleString()} ${t("个线程")}${state.paused ? ` · ${t("视图已暂停")}` : ""}`;
}

function activityFooter(state: ViewState, message: string, explanation: string): string {
  const snapshot = state.snapshot;
  return `<footer class="activity-footer"><span class="activity-source" title="${e(explanation)}">${e(message)}</span>${state.page === "processes" ? `<div class="activity-totals"><span class="total-cpu">CPU <b>${snapshot?.cpu?.ready ? percent(snapshot.cpu.usage) : "—"}</b></span><span class="total-memory">${t("内存")} <b>${snapshot?.memory?.total ? formatMemory(snapshot.memory.used) : "—"}</b></span><span class="total-power">${t("功率")} <b>${snapshot?.battery?.systemWatts != null ? snapshot.battery.systemWatts.toFixed(1) + " W" : "—"}</b></span></div>` : ""}</footer>`;
}

function tableSpacer(height: number, columns: number, key: string): string {
  return `<tr class="virtual-spacer" data-row-key="${key}" aria-hidden="true"><td colspan="${columns}" style="height:${height}px"></td></tr>`;
}

const processTable = cachedMarkup(state => {
  const allRows = orderedProcesses(state);
  const window = windowRows(allRows, state.tableWindow), rows = window.rows;
  const columns: [string, string][] = [["名称", "name"], ["% CPU", "cpu"], ["% GPU", ""], ["内存", "memory"], ["功率", ""], ["线程", "threads"], ["CPU 时间", ""], ["PID", "pid"], ["用户", "userId"]];
  const processHeaders = columns.map(([label, key]) => `<th scope="col"${key ? ` aria-sort="${state.processOrder?.key === key ? state.processOrder.descending ? "descending" : "ascending" : "none"}"` : ""}>${key ? `<button class="activity-sort" data-process-sort="${key}">${t(label)}${state.processOrder?.key === key ? `<span class="sort-direction ${state.processOrder.descending ? "descending" : "ascending"}">${icon("chevron")}</span>` : ""}</button>` : t(label)}</th>`).join("");
  return rows.length ? `<table class="process-table activity-table${state.tableWindow ? " is-windowed" : ""}" aria-rowcount="${allRows.length + 1}" data-window-start="${window.start}"><thead><tr>${processHeaders}</tr></thead><tbody>${state.tableWindow ? tableSpacer(window.before, 9, "before") : ""}${rows.map((p, index) => `<tr aria-rowindex="${window.start + index + 2}" data-row-key="${p.pid}:${e(p.identity)}" class="${state.selectedProcess === p.pid ? "selected" : ""} ${(window.start + index) % 2 ? "striped" : ""}" data-select-process="${p.pid}"><td><button class="connection-process" data-select-process="${p.pid}" aria-pressed="${state.selectedProcess === p.pid}" title="${e(p.path)}">${processIcon(p, state)}<span>${e(p.name)}</span></button></td><td>${p.cpuReady ? p.cpu.toFixed(1) : "—"}</td><td class="connection-unavailable">—</td><td>${formatMemory(p.memory)}</td><td class="connection-unavailable">—</td><td>${p.threads}</td><td class="connection-unavailable">—</td><td>${p.pid}</td><td>${p.userId}</td></tr>`).join("")}${state.tableWindow ? tableSpacer(window.after, 9, "after") : ""}</tbody></table>` : empty(t("没有匹配的进程"), t("试试其他名称，或查看完整进程列表。"));
}, state => [state.processes, state.query, state.processScope, state.onlyMine, state.sort,
  state.processOrder?.key, state.processOrder?.descending, state.selectedProcess, isMac(state), i18n.language, state.tableWindow?.start, state.tableWindow?.count]);

function processView(state: ViewState): string {
  if (state.processError) return empty(t("无法读取进程"), state.processError, `<button data-action="refresh">${t("重试")}</button>`);
  if (!state.processes) return empty(t("正在读取进程"), t("进程采集独立于窗口绘制运行。"));
  const selected = state.processes.items.find(p => p.pid === state.selectedProcess);
  const missing = t("当前采集源未提供单进程 GPU、功率或 CPU 时间；用户列显示 UID。");
  const history = state.processHistory || [];
  const end = Date.parse(state.processes.updatedAt);
  const detail = state.selectedProcess == null ? "" : `<aside class="process-detail" aria-label="${t("进程详情")}">
    <div class="process-detail-heading">${selected ? processIcon(selected, state) : ""}<div><strong>${selected ? e(selected.name) : t("进程不可用")}</strong><p>PID ${state.selectedProcess}${selected ? ` · UID ${selected.userId}` : ""}</p></div><button class="icon-button" data-action="close-process" aria-label="${t("关闭详情")}" title="${t("关闭详情")}">${icon("close")}</button></div>
    ${state.processAction?.pid === state.selectedProcess && state.processAction.message ? `<p class="process-action-message ${state.processAction.error ? "error" : ""}" role="${state.processAction.error ? "alert" : "status"}">${e(state.processAction.message)}</p>` : ""}
    ${selected ? `<div class="process-detail-actions"><button data-action="quit-process" ${state.processAction?.busy || !selected.identity || !["darwin", "linux"].includes(state.info?.machine.platform || "") ? "disabled" : ""} title="${state.info?.machine.platform === "windows" ? t("Windows 暂不支持请求正常退出，请使用强制退出。") : t("请求进程正常退出")}">${t("退出")}</button><button data-action="force-quit-process" ${state.processAction?.busy || !selected.identity ? "disabled" : ""} class="destructive">${t("强制退出")}</button><button disabled aria-label="${t("可执行文件路径")}">${icon("folder")}</button></div>
    <div class="process-metrics"><div class="process-cpu"><header><span>CPU</span><strong>${selected.cpuReady ? percent(selected.cpu) : "—"}</strong></header>${dots(history, ["cpu"], end, 60, t("进程 CPU 历史"), 100, { columns: 42, rows: 6, radius: 1.7 })}</div><div class="process-memory"><header><span>${t("内存")}</span><strong>${formatMemory(selected.memory)}</strong></header>${dots(history, ["memory"], end, 60, t("进程内存历史"), 0, { columns: 42, rows: 6, radius: 1.7 })}</div></div>
    <h2>${t("详细信息")}</h2>${facts([[t("CPU 时间"), "—"], [t("启动时间"), "—"], [t("状态"), "—"], [t("线程"), String(selected.threads)], [t("读取"), selected.ioReady ? formatRate(selected.readRate) : "—"], [t("写入"), selected.ioReady ? formatRate(selected.writeRate) : "—"]])}
    <h2>${t("父进程")}</h2><div class="process-parent"><span class="process-system-icon">${icon("settings")}</span><span>—</span></div>
    <h2>${t("命令")}</h2><p class="process-path" title="${e(selected.path)}">${e(selected.path || "—")}</p>` : `<p class="muted">${t("进程可能已退出或不再可见。")}</p>`}
  </aside>`;
  const table = processTable(state);
  return `<div class="activity-workspace process-layout ${state.selectedProcess == null ? "" : "has-detail"}"><div class="activity-scroll process-list">${table}</div>${detail}</div>${activityFooter(state, `${state.processes.skipped} ${t("个因权限或已退出未读取")}`, missing)}`;
}

export function connectionSubtitle(state: ViewState): string {
  if (!state.connections) return t("正在读取网络连接");
  const total = state.connections.items.length;
  const count = state.query.trim() ? t("{{shown}} / {{total}} 个连接", { shown: connectionRows(state.connections.items, state.processes?.items || [], state.query).length, total }) : t("{{count}} 个连接", { count: total });
  return `${count} · ${state.paused ? t("视图已暂停") : timeLabel(state.connections.updatedAt)}`;
}

function connectionStatus(status: string): string {
  switch (status) {
    case "ESTABLISHED": return t("已连接");
    case "LISTEN": return t("监听中");
    case "SYN_SENT": case "SYN_RECV": return t("连接中");
    case "TIME_WAIT": case "CLOSE_WAIT": case "FIN_WAIT1": case "FIN_WAIT2": case "LAST_ACK": case "CLOSING": return t("关闭中");
    case "CLOSED": case "CLOSE": return t("已关闭");
    default: return status || "—";
  }
}

const connectionView = cachedMarkup(state => {
  if (state.connectionError && !state.connections) return empty(t("无法读取连接"), state.connectionError, `<button data-action="refresh">${t("重试")}</button>`);
  if (!state.connections) return empty(t("正在读取网络连接"), t("读取当前权限可见的 TCP / UDP 套接字。"));
  const allRows = connectionRows(state.connections.items, state.processes?.items || [], state.query, state.connectionSort);
  const window = windowRows(allRows, state.tableWindow), rows = window.rows;
  const missing = isMac(state) ? t("延迟为 TCP RTT；速率取两次采样差值。— 表示无样本或无法匹配。") : t("当前采集源未提供单连接延迟、速率及累计流量。");
  const headers = [t("进程"), t("主机"), t("服务"), t("协议"), t("状态"), t("延迟"), t("下载"), t("上传"), t("已接收"), t("已发送")];
  const keys: ConnectionSortKey[] = ["process", "host", "port", "protocol", "state", "rtt", "download", "upload", "received", "sent"];
  const headerCells = headers.map((header, i) => {
    const key = keys[i];
    const direction = state.connectionSort && state.connectionSort.key === key ? state.connectionSort.direction : "none";
    return `<th scope="col" aria-sort="${direction}"><button class="connection-sort" data-connection-sort="${key}">${header}${direction === "none" ? "" : `<span class="sort-direction ${direction}" aria-hidden="true">${icon("chevron")}</span>`}</button></th>`;
  }).join("");
  const localLabel = t("本地地址"), remoteLabel = t("远端地址"), unavailableLabel = t("暂不可用");
  const hints = [t("TCP 平均往返时间，不是 ping 延迟。"), t("本次采样窗口内的平均接收速率。"), t("本次采样窗口内的平均发送速率。"), t("系统报告的该套接字累计接收字节数。"), t("系统报告的该套接字累计发送字节数。")];
  const statusLabels = new Map(rows.map(row => [row.state, ""]));
  for (const status of statusLabels.keys()) statusLabels.set(status, connectionStatus(status));
  const unavailableCell = `<td class="connection-unavailable" title="${e(missing)}" aria-label="${unavailableLabel}">—</td>`;
  const metricCell = (value: number | null | undefined, format: (n: number) => string, hint: string) => value == null
    ? unavailableCell : `<td class="connection-metric" title="${e(hint)}">${format(value)}</td>`;
  return `${state.connectionError ? notice(`${t("更新中断 · 上次数据")} · ${state.connectionError}`) : ""}${state.connections.warning ? notice(state.connections.warning) : ""}${rows.length ? `<div class="activity-workspace"><div class="activity-scroll"><table class="connection-table activity-table${state.tableWindow ? " is-windowed" : ""}" aria-rowcount="${allRows.length + 1}" data-window-start="${window.start}" aria-label="${t("连接")}"><thead><tr>${headerCells}</tr></thead><tbody>${state.tableWindow ? tableSpacer(window.before, 10, "before") : ""}${rows.map((c, index) => {
    const endpoint = splitEndpoint(c.remote || c.local);
    const addresses = `${localLabel}: ${c.local} · ${remoteLabel}: ${c.remote || "—"} · ${c.family}`;
    const metrics = c.metrics;
    const cells = metricCell(metrics?.rtt, n => `${n.toFixed(1)} ms`, hints[0]!)
      + metricCell(metrics?.download, formatRate, hints[1]!)
      + metricCell(metrics?.upload, formatRate, hints[2]!)
      + metricCell(metrics?.received, formatBytes, hints[3]!)
      + metricCell(metrics?.sent, formatBytes, hints[4]!);
    return `<tr data-row-key="connection-${window.start + index}" aria-rowindex="${window.start + index + 2}" class="${(window.start + index) % 2 ? "striped" : ""}"><td><button class="connection-process" data-process="${c.pid}" title="${e(c.process)} · PID ${c.pid}${c.path ? ` · ${e(c.path)}` : ""}">${c.path ? appIcon(c.path, isMac(state)) : `<span class="app-icon process-system-icon">${icon("settings")}</span>`}<span>${e(c.process || String(c.pid))}</span></button></td><td title="${e(addresses)}">${e(endpoint.host || "—")}</td><td class="connection-secondary" title="${e(c.remote || c.local)}">${e(endpoint.port || "—")}</td><td>${e(c.protocol)}</td><td class="connection-secondary" title="${e(c.state)}">${e(statusLabels.get(c.state)!)}</td>${cells}</tr>`;
  }).join("")}${state.tableWindow ? tableSpacer(window.after, 10, "after") : ""}</tbody></table></div></div>` : empty(t("没有匹配的连接"), t("当前权限下没有匹配的 TCP / UDP 记录，可调整搜索或刷新。"))}${activityFooter(state, missing, isMac(state) ? t("来源：本机 lsof 与 nettop。仅匹配 PID、协议和两端地址唯一对应的套接字，不提权、不解析远端域名、不主动探测。") : t("来源：Windows 系统连接表或 Linux procfs。只包含当前权限可见的套接字；进程归属可能不可见。不提权、不解析远端域名、不发起请求。"))}`;
}, state => [state.connections, state.processes, state.query, state.connectionSort?.key,
  state.connectionSort?.direction, state.connectionError, isMac(state), i18n.language, state.tableWindow?.start, state.tableWindow?.count]);

function groupApps(processes: Process[]): AppUsage[] {
  const groups = new Map<string, AppUsage>();
  for (const p of processes) {
    if (!p.app) continue;
    const app = groups.get(p.app) || { name: p.app.split(/[\\/]/).at(-1)!.replace(/\.app$/, ""), path: p.app, processes: 0, cpu: 0, cpuReady: true, memory: 0 };
    app.processes++; app.cpu += p.cpu; app.memory += p.memory; app.cpuReady &&= p.cpuReady; groups.set(p.app, app);
  }
  return [...groups.values()];
}

function appsView(state: ViewState): string {
  const mode = isMac(state) ? `<div class="app-mode-toolbar"><div class="segmented"><button class="${!state.appMode ? "selected" : ""}" aria-pressed="${!state.appMode}" data-action="running-apps">${t("运行中的 App")}</button><button class="${state.appMode ? "selected" : ""}" aria-pressed="${state.appMode}" data-action="installed-apps">${t("已安装的 App")}</button></div></div>` : "";
  if (isMac(state) && state.appMode) return mode + installedAppsView(state);
  if (state.processError) return mode + empty(t("无法读取 App"), state.processError, `<button data-action="refresh">${t("重试")}</button>`);
  if (!state.processes) return mode + empty(t("正在读取运行中的 App"), t("将同一个 App 包中的进程归并显示。"));
  const apps = groupApps(state.processes.items).filter(app => `${app.name} ${app.path}`.toLowerCase().includes(state.query.toLowerCase()));
  apps.sort((a, b) => state.sort === "name" ? a.name.localeCompare(b.name) : state.sort === "cpu" ? b.cpu - a.cpu : b.memory - a.memory);
  return mode + `<div class="result-summary"><span>${apps.length} ${t("个正在运行的 App")}</span><span>${(isMac(state) ? t("按 App 包路径归并 · 不含未启动的 App") : t("按可执行文件路径归并 · 仅含运行中的进程"))}</span></div>${apps.length ? `<div class="app-list"><div class="app-row app-head"><span>App</span><span>${t("物理占用")}</span><span>CPU</span><span>${t("进程")}</span></div>${apps.map(app => `<button class="app-row" data-app="${e(app.path)}">${appIcon(app.path, isMac(state))}<span class="app-name"><strong>${e(app.name)}</strong><small>${e(app.path)}</small></span><span>${formatMemory(app.memory)}</span><span>${app.cpuReady ? percent(app.cpu) : t("采样中")}</span><span>${app.processes}</span><span class="card-chevron">${icon("chevron")}</span></button>`).join("")}</div>` : empty(t("没有匹配的 App"), t("试试其他名称，或查看完整进程列表。"))}${note((isMac(state) ? t("只读查看运行状态。物理占用来自各进程 phys_footprint 之和；此页面不执行卸载或删除应用数据。") : t("只读查看运行状态。内存为各进程 RSS 之和，共享页可能重复计入；此页面不执行卸载或删除。")))}`;
}

function appScanCoverage(review: AppReview): string {
  if (!review.scanIssues.length) return "";
  const issueLabel = (kind: string) => kind === "permission" ? t("没有访问权限") : kind === "changed" ? t("扫描期间位置已变化") : t("读取失败");
  const permission = review.scanIssues.some(issue => issue.kind === "permission") ? `<div class="disk-access-prompt" role="status"><div><strong>${t("部分位置需要访问权限")}</strong><p>${t("开启完全磁盘访问权限可扩大搜索范围。在系统设置中允许 RockshinOps；若列表里没有，点“+”添加。开启后退出并重新打开 RockshinOps，再重新搜索。部分系统目录仍可能受限。")}</p></div><button class="secondary-button" data-action="open-full-disk-access">${t("打开完全磁盘访问权限")}</button></div>` : "";
  return permission + `<details class="uninstall-coverage"><summary><strong>${t("扫描覆盖范围受限")}</strong><span>${t("未读取的位置：{{count}} 个", { count: review.scanIssues.length })}</span></summary><p>${t("这是整个搜索范围的访问情况，不是此 App 的残留数量。已找到的项目仍可核对清理；未读取的位置中可能还有关联文件。")}</p><ul>${review.scanIssues.map(issue => `<li><span class="path">${e(issue.path)}</span><small>${e(issueLabel(issue.kind))} · ${e(issue.error)}</small></li>`).join("")}</ul></details>`;
}

function installedAppsView(state: ViewState): string {
  const message = state.appMessage ? notice(state.appMessage) : "";
  if (state.appBusy && !state.appReview) return message + empty(t("正在读取 App"), t("请稍候。"));
  if (state.appReview) {
    const review = state.appReview;
    const selected = state.appSelection.length;
    const allSelected = review.files.length > 0 && selected === review.files.length;
    return message + `<button class="cleanup-action uninstall-back" data-action="installed-apps" ${state.appBusy ? "disabled" : ""}>${t("返回应用列表")}</button><section class="uninstall-review"><div class="uninstall-heading">${appIcon(review.app.path, true)}<div><h2>${e(review.app.name)}</h2><p class="path">${e(review.bundleId || review.app.path)}</p></div></div><p class="uninstall-guidance">${t("请逐项核对后选择。名称匹配可能属于共享数据；搜索无法保证找到所有残留。移除前请退出 App。")}</p>${state.appSearching ? `<div class="uninstall-progress" role="status"><span class="loading-dot" aria-hidden="true"></span><span>${t("正在继续搜索关联文件…")}</span><button class="cleanup-action" data-action="stop-app-search">${t("停止搜索")}</button></div>` : ""}${appScanCoverage(review)}<div class="uninstall-list-heading"><label><input type="checkbox" data-app-select-all="true" aria-label="${t("全选关联项目")}" ${allSelected ? "checked" : ""} ${state.appBusy && !state.appSearching ? "disabled" : ""}><span class="uninstall-list-title">${t("关联项目")}</span></label><span>${t("已选择 {{selected}} / {{total}} 项", { selected, total: review.files.length })}</span></div><div class="uninstall-list">${review.files.map(file => `<label><input type="checkbox" data-app-file="${e(file.path)}" ${state.appBusy && !state.appSearching ? "disabled" : ""} ${state.appSelection.includes(file.path) ? "checked" : ""}><span><strong class="path">${e(file.path)}</strong><small>${e(t(file.reason))}</small></span></label>`).join("")}</div><div class="uninstall-actions"><span>${state.appSearching ? t("搜索完成或停止后可移入废纸篓。") : t("所选项目将移入废纸篓。")}</span><button class="primary-button" data-action="remove-app-files" ${selected && !state.appBusy ? "" : "disabled"}>${t("移入废纸篓")}</button></div></section>`;
  }
  if (!state.installedApps) return message + empty(t("无法读取已安装的 App"), t("请重试。"));
  const apps = state.installedApps.filter(app => `${app.name} ${app.path}`.toLowerCase().includes(state.query.toLowerCase()));
  return message + `<div class="installed-hint"><span>${t("选择 App 后可查看应用包和可能的关联文件。")}</span><button class="cleanup-action" data-action="open-full-disk-access">${t("打开完全磁盘访问权限")}</button></div>` + (apps.length ? `<div class="app-list installed-list">${apps.map(app => `<button data-review-app="${e(app.path)}">${appIcon(app.path, true)}<span><strong>${e(app.name)}</strong><small class="path">${e(app.path)}</small></span>${icon("chevron")}</button>`).join("")}</div>` : empty(t("没有匹配的 App"), t("试试其他名称或路径。")));
}

function cleanupLanding(state: ViewState): string {
  const disk = state.snapshot!.disk;
  const volume = disk.volume;
  const capacity = !disk.capacityError && volume.total > 0;
  const unavailableRatio = capacity ? 1 - volume.available / volume.total : 0;
  const targets = state.info!.scanTargets;
  return `<div class="flex flex-col gap-4">
    <section class="storage-summary disk" aria-labelledby="storage-title">
      <div class="flex items-center gap-2 text-text"><span class="content-icon">${icon("disk")}</span><h2 id="storage-title">${t("磁盘空间")}</h2><span class="ml-auto path text-muted">${e(volume.name || volume.mount)}</span></div>
      ${disk.capacityError ? `<p class="mt-3 text-small text-error" role="status">${e(disk.capacityError)}</p>` : ""}
      <dl class="storage-values">
        <div><dt>${t("剩余可写空间")}</dt><dd class="storage-available">${capacity ? formatCapacity(volume.available) : t("不可用")}</dd></div>
        <div><dt>${t("已占用及保留")}</dt><dd>${capacity ? formatCapacity(volume.total - volume.available) : "—"}</dd></div>
        <div><dt>${t("容量")}</dt><dd>${capacity ? formatCapacity(volume.total) : "—"}</dd></div>
      </dl>
      ${capacity ? `<div class="capacity-track" aria-hidden="true"><i style="transform:scaleX(${unavailableRatio})"></i></div>` : ""}
      <p class="mt-3 text-caption text-muted">${t("当前用户可写的剩余容量")} · ${t("用户主目录所在卷")}</p>
    </section>
    <div class="cleanup-workspaces">
      <section class="cleanup-workspace" aria-labelledby="explore-title">
        <div class="flex items-center gap-2"><span class="content-icon">${icon("folder")}</span><h2 id="explore-title">${t("透视磁盘")}</h2></div>
        <h3 class="mt-5 text-[20px] font-semibold tracking-tight">${t("先了解空间去了哪里")}</h3>
        <p class="mt-2 text-small leading-relaxed text-muted">${t("从磁盘根目录开始，按文件夹逐层查看空间分布。")}</p>
        <dl class="fact-list my-5">
          <div><dt>${t("扫描范围")}</dt><dd>${t("系统磁盘")}</dd></div>
          <div><dt>${t("分析方式")}</dt><dd>${t("只读取文件大小")}</dd></div>
          <div><dt>${t("清理方式")}</dt><dd>${t("核对后移入废纸篓")}</dd></div>
        </dl>
        <div class="mt-auto pt-2"><button class="primary-button" data-action="space-start">${t("透视磁盘")}${icon("chevron")}</button></div>
      </section>
      <section class="cleanup-workspace" aria-labelledby="targets-title">
        <div class="flex items-center gap-2"><span class="content-icon">${icon("cleanup")}</span><h2 id="targets-title">${t("缓存与废纸篓")}</h2></div>
        <p class="mt-3 text-small leading-relaxed text-muted">${t("分析下列配置目录的文件大小。文件内容不会被读取。")}</p>
        <ul class="cleanup-targets">${targets.map(target => `<li><span class="inline-flex shrink-0 text-muted">${icon("folder")}</span><div class="min-w-0"><strong class="block text-small font-medium">${e(t(target.name))}</strong><span class="path block text-muted">${e(target.path)}</span></div></li>`).join("")}</ul>
        <div class="mt-auto pt-2"><button class="secondary-button" data-action="cleanup-targets">${t("查看缓存与废纸篓")}${icon("chevron")}</button></div>
      </section>
    </div>
    <p class="text-caption leading-relaxed text-muted">${t("分析只读取文件大小。选中的项目会移入系统废纸篓；清空废纸篓将永久删除所有内容。缓存可能仍被 App 使用。")}</p>
  </div>`;
}

function cleanupView(state: ViewState): string {
  const intro = notice(t("分析只读取文件大小。选中的项目会移入系统废纸篓；清空废纸篓将永久删除所有内容。缓存可能仍被 App 使用。"));
  if (state.scanning) return intro + `<div class="empty-state" role="status" aria-live="polite"><span class="scan-spinner" aria-hidden="true"></span><strong>${t("正在扫描本机空间")}</strong>${state.spaceMode ? `<strong class="scan-progress" data-scan-progress>${t("正在统计扫描项目…")}</strong>` : ""}<p>${t(state.spaceMode ? "进度按项目数量计算，不代表剩余时间。文件较多时统计和扫描可能需要较长时间；无法读取的项目会在结果中标明。" : "逐项读取配置的目录。遇到权限限制会标注，达到时间或数量上限会返回已读部分。")}</p>${state.spaceMode && state.spaceHistory.length ? `<p class="path">${e(state.spaceHistory.at(-1)!)}</p>` : ""}</div>`;
  if (state.scanError) return intro + empty(t("扫描未完成"), state.scanError, `<button data-action="scan">${t("重新扫描")}</button>`);
  if (!state.scan) return cleanupLanding(state);
  return intro + `<div class="space-toolbar">${state.spaceMode ? `<button class="cleanup-action" data-action="space-back" ${state.spaceHistory.length ? "" : "disabled"}>${t("返回上一级")}</button><button class="cleanup-action" data-action="cleanup-targets">${t("查看缓存与废纸篓")}</button>` : `<button class="cleanup-action" data-action="space-start">${t("透视磁盘")}</button>`}</div>` + `${state.cleanupMessage ? notice(state.cleanupMessage) : ""}${!state.scan.complete ? notice(state.scan.warning || t("部分目录无法读取。以下数据仅代表成功读取的文件。")) : ""}<div class="result-summary"><span>${state.scan.complete ? t("扫描完成") : t("扫描结果不完整")}</span><span>${timeLabel(state.scan.finishedAt)} ${t("· 逻辑文件大小，不代表实际可释放空间")}</span></div><div class="scan-groups">${state.scan.groups.map(group => `<section class="scan-group"><div class="scan-group-heading"><div><h2>${e(t(group.name))}</h2><p class="path">${e(group.path)}</p></div><div><strong>${formatCapacity(group.bytes)}</strong><span>${group.files} ${t("个文件")}</span></div></div>${group.error ? notice(group.error) : ""}${group.skipped ? note(`${group.skipped} ${t("项因权限或读取错误跳过。")}`) : ""}${group.error || group.skipped ? note(t("此目录扫描不完整，清理不可用。")) : ""}<div class="space-list">${group.items.map(item => `<div><span title="${e(item.path)}">${icon("folder")}<strong>${e(item.name)}</strong>${state.spaceMode ? `<i class="space-share" style="--share:${group.bytes ? Math.min(100, item.bytes / group.bytes * 100) : 0}%"></i>` : ""}</span>${state.spaceMode && item.directory ? `<button class="cleanup-action" data-space-path="${e(item.path)}">${t("打开")}</button>` : ""}<span>${item.scanned ? `${item.files} ${t("项")}` : t("未扫描")}</span><span>${item.scanned ? formatCapacity(item.bytes) : "—"}</span>${group.name !== "废纸篓" && !(state.info?.scanTargets || []).some(target => target.name === "废纸篓" && (item.path === target.path || item.path.startsWith(target.path + (target.path.includes("\\") ? "\\" : "/")))) ? `<button class="cleanup-action" data-cleanup-path="${e(item.path)}" ${!!group.error || group.skipped > 0 || state.cleanupBusy ? "disabled" : ""}>${t("移入废纸篓")}</button>` : ""}</div>`).join("") || `<p class="muted">${t("未读取到普通文件")}</p>`}</div>${group.name === "废纸篓" ? `<button class="cleanup-action" data-empty-trash="${e(group.path)}" ${!!group.error || group.skipped > 0 || state.cleanupBusy ? "disabled" : ""}>${t("清空废纸篓")}</button>` : ""}</section>`).join("")}</div>${note(t("不跟随符号链接。目录按逻辑大小汇总；硬链接、APFS 克隆和多次挂载的路径可能重复计入。"))}`;
}

function aboutView(state: ViewState): string {
  const i = state.info!, m = i.machine, s = state.snapshot!, disk = s.disk;
  const name = m.name || m.model || t("本机");
  const text = (value: string) => value ? e(value) : unavailable(t("不可用"));
  const cores = m.physicalCPUs > 0 ? String(m.physicalCPUs) + (m.performanceCores != null && m.efficiencyCores != null ? ` (${t("核心分布", { performance: m.performanceCores, efficiency: m.efficiencyCores })})` : "") : unavailable(t("不可用"));
  const bootTime = new Date(new Date(s.updatedAt).getTime() - s.uptimeSeconds * 1000);
  const section = (title: string, symbol: string, kind: string, rows: [string, string][]) => `<section class="about-section" data-kind="${kind}"><h2><span class="content-icon">${icon(symbol)}</span>${t(title)}</h2>${facts(rows)}</section>`;
  const displays: [string, string][] = i.displays.length ? i.displays.map(display => [display.primary ? t("主显示器") : display.name || t("显示器"), `${display.width} × ${display.height} @${display.scale}x`]) : [[t("显示器"), unavailable(t("不可用"))]];
  return `<div class="about-page">
    <section class="about-hero" aria-label="${t("这台电脑")}">
      <span class="machine-icon">${isMac(state) ? '<img src="appicon://localhost/computer" alt="">' : icon("about")}</span>
      <div class="about-identity"><h2>${e(name)}</h2><p>${text(m.chip)} · ${m.memory > 0 ? formatMemory(m.memory) : t("不可用")}</p>
        <div class="about-status"><span>${text(m.os)}${m.osBuild ? ` (${e(m.osBuild)})` : ""}</span><span>${t("已运行")} ${duration(s.uptimeSeconds)}</span></div>
      </div>
    </section>
    ${m.error ? notice(m.error) : ""}
    <div class="about-grid">
      ${section("硬件", "cpu", "hardware", [[t("型号"), text(m.name)], [t("型号标识"), text(m.model)], [t("芯片"), text(m.chip)], [t("CPU 核心"), cores], [t("GPU 核心"), m.gpuCores == null ? unavailable(t("不可用")) : String(m.gpuCores)], [t("内存"), m.memory > 0 ? formatMemory(m.memory) : unavailable(t("不可用"))], [t("架构"), text(m.architecture)], [t("序列号"), text(m.serialNumber)]])}
      ${section("软件", "about", "software", [[t("操作系统"), text(m.os)], [t("构建版本"), text(m.osBuild)], [t("内核"), text(m.kernel)], [t("电脑名称"), text(m.hostname)], [t("启动时间"), e(bootTime.toLocaleString(i18n.language, { dateStyle: "medium", timeStyle: "short" }))], [t("运行时间"), duration(s.uptimeSeconds)], [t("进程"), String(s.processCount)], ["Go", e(i.goVersion)]])}
      ${section("显示器", "about", "display", displays)}
      ${section("存储", "disk", "storage", [[disk.volume.name || disk.volume.mount, disk.capacityError ? unavailable(disk.capacityError) : t("可写容量与总量", { available: formatCapacity(disk.volume.available), total: formatCapacity(disk.volume.total) })]])}
    </div>
  </div>`;
}

export function renderBody(state: ViewState): string {
  if (state.page === "settings" || state.page === "menubar") return "";
  if (!state.snapshot || !state.info) return empty(t("等待本机数据"), t("点击右上角刷新重新连接采集服务。"));
  switch (state.page) {
    case "overview": return overview(state);
    case "cpu": return cpuView(state);
    case "memory": return memoryView(state);
    case "gpu": return gpuView(state);
    case "disk": return diskView(state);
    case "network": return networkView(state);
    case "energy": return energyView(state);
    case "processes": return processView(state);
    case "connections": return connectionView(state);
    case "apps": return appsView(state);
    case "cleanup": return cleanupView(state);
    case "about": return aboutView(state);
  }
}
