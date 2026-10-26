import { t } from "./i18n";
import type { AppInfo, Snapshot, Trend, ProcessList, Connections, ScanResult, AppUsage, Process } from "./mygo";
import { capacityParts, duration, escapeHTML as e, formatCapacity, formatMemory, formatRate, memoryParts, percent, rateParts, timeLabel } from "./format";
import { dots, lineChart, type TrendKey } from "./charts";
import { icon } from "./icons";
import { sortControl } from "./dropdown";
import { type SettingsState } from "./settings";

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
  { id: "about", title: "关于本机", group: "系统" },
  { id: "settings", title: "设置", group: "系统" },
] as const;
export type Page = (typeof pages)[number]["id"];
export const metricPages: Page[] = ["overview", "cpu", "memory", "gpu", "disk", "network", "energy"];

export interface ViewState extends SettingsState {
  page: Page;
  info: AppInfo | null;
  snapshot: Snapshot | null;
  history: Trend[];
  range: number;
  paused: boolean;
  processes: ProcessList | null;
  connections: Connections | null;
  scan: ScanResult | null;
  query: string;
  sort: string;
  onlyMine: boolean;
  busy: boolean;
  processError: string;
  connectionError: string;
  scanError: string;
  scanning: boolean;
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
    return `${heading}<button type="button" class="nav-item" data-page="${page.id}" ${page.id === "overview" ? 'aria-current="page"' : ""}><span class="nav-icon">${icon(page.id)}</span><span>${t(page.title)}</span><small id="nav-${page.id}"></small></button>`;
  }).join("");
}

export function actionControls(state: ViewState): string {
  if (state.page === "settings") return `<span class="settings-shortcut">${isMac(state) ? "⌘," : "Ctrl+,"}</span>`;
  const ranges = [[60, t("1 分钟")], [300, t("5 分钟")], [900, t("15 分钟")], [3600, t("1 小时")]] as const;
  const history = metricPages.includes(state.page) ? `<div class="segmented" role="group" aria-label="${t("历史时间范围")}">${ranges.map(([seconds, label]) => `<button data-range="${seconds}" class="${state.range === seconds ? "selected" : ""}" aria-pressed="${state.range === seconds}">${label}</button>`).join("")}</div><button class="icon-button pause-button ${state.paused ? "selected" : ""}" data-action="pause" aria-label="${state.paused ? t("恢复实时视图") : t("暂停实时视图")}" title="${state.paused ? t("恢复实时视图") : t("暂停视图，菜单栏继续采集")}">${icon(state.paused ? "play" : "pause")}</button>` : "";
  return `${history}<button class="icon-button" data-action="refresh" ${state.busy ? 'aria-disabled="true" aria-busy="true"' : ""} aria-label="${state.page === "cleanup" ? t("重新扫描") : t("立即刷新")}" title="${state.page === "cleanup" ? t("重新扫描") : t("立即刷新")}">${icon("refresh")}</button>`;
}

export function filterControls(state: ViewState): string {
  if (!["processes", "connections", "apps"].includes(state.page)) return "";
  const sorting = state.page === "connections" ? "" : sortControl(state.sort, state.page === "processes");
  return `<label class="search-field">${icon("search")}<input id="search" type="search" placeholder="${state.page === "connections" ? t("搜索进程、地址或协议") : t("搜索名称或 PID")}" aria-label="${t("搜索")}${t(pages.find(p => p.id === state.page)!.title)}" value="${e(state.query)}" autocomplete="off" spellcheck="false"/></label>${sorting}${state.page === "processes" ? `<label class="checkbox-label"><input id="only-mine" type="checkbox" ${state.onlyMine ? "checked" : ""}/>${t("仅我的进程")}</label>` : ""}`;
}

function card(state: ViewState, page: Page, color: string, title: string, label: string, main: string, items: [string, string][], keys: TrendKey[], max = 100, status = ""): string {
  return `<button class="metric-card ${color}" data-page="${page}" type="button"><div class="card-heading"><span class="card-icon">${icon(page === "energy" ? "battery" : page)}</span><h2>${e(title)}</h2><span class="card-chevron">${icon("chevron")}</span></div><p class="metric-label">${e(label)}</p><div class="metric-value">${main}${status}</div>${stats(items)}${dots(state.history, keys, new Date(state.snapshot!.updatedAt).getTime(), state.range, title + t("历史"), max)}</button>`;
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
  return state.history.filter(p => new Date(p.at).getTime() >= cutoff && p[key] != null).map(p => p[key]!);
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
  return `<section class="summary-card memory"><h2>${icon("memory")}${t("内存构成")}</h2>${m.error ? unavailable(m.error) : `<div class="memory-composition"><div class="memory-ring"><svg viewBox="0 0 120 120" aria-hidden="true">${segments.join("")}</svg><div><strong>${percent(m.used / m.total * 100)}</strong><span>${t("已使用")}</span></div></div><dl class="ring-legend">${items.map(([name, bytes, color]) => `<div><dt><i class="legend-dot ring-${color}"></i>${name}</dt><dd>${formatMemory(bytes)}</dd></div>`).join("")}</dl></div>`}${note(t("可用空间包含可回收部分；缓存单独列于内存详情。"))}</section>`;
}

function appSummary(state: ViewState, apps: AppUsage[]): string {
  const max = Math.max(1, ...apps.map(app => app.memory));
  return `<section class="summary-card memory"><h2>${icon("apps")}${t("App 内存占用")}</h2><div class="rank-list">${apps.slice(0, 5).map(app => `<button class="rank-item" data-app="${e(app.path)}">${appIcon(app.path, isMac(state))}<span class="rank-copy"><span class="rank-heading"><strong>${e(app.name)}</strong><span>${formatMemory(app.memory)}</span></span><span class="rank-track"><i style="transform:scaleX(${app.memory / max})"></i></span></span></button>`).join("") || unavailable(t("等待进程采样"))}</div>${note((isMac(state) ? t("按 App 包归并进程的物理内存占用。") : t("按可执行文件路径归并进程的常驻内存占用。")))}</section>`;
}

function cpuSummary(processes: Process[]): string {
  const max = Math.max(1, ...processes.map(p => p.cpu));
  return `<section class="summary-card energy"><h2>${icon("energy")}${t("CPU 活跃进程")}</h2><div class="rank-list">${processes.slice(0, 5).map(p => `<button class="rank-item" data-process="${p.pid}"><span class="app-letter">${e(p.name.slice(0, 1).toUpperCase())}</span><span class="rank-copy"><span class="rank-heading"><strong>${e(p.name)}</strong><span>${p.cpuReady ? percent(p.cpu) : t("采样中")}</span></span><span class="rank-track"><i style="transform:scaleX(${p.cpu / max})"></i></span></span></button>`).join("") || unavailable(t("等待进程采样"))}</div>${note(t("单核占满为 100%；这不是 App 功耗排名。"))}</section>`;
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
  return details + notice(t("当前未接入 App 级瓦特计量。CPU 活跃进程有助于定位负载，但不能据此换算各 App 的功耗。"));
}

function orderedProcesses(state: ViewState): Process[] {
  const query = state.query.toLowerCase();
  const result = (state.processes?.items || []).filter(p => (!state.onlyMine || p.own) && `${p.name} ${p.pid} ${p.path}`.toLowerCase().includes(query));
  result.sort((a, b) => state.sort === "name" ? a.name.localeCompare(b.name) : (state.sort === "memory" ? b.memory - a.memory : state.sort === "readRate" ? b.readRate - a.readRate : state.sort === "writeRate" ? b.writeRate - a.writeRate : b.cpu - a.cpu) || a.pid - b.pid);
  return result;
}

function processView(state: ViewState): string {
  if (state.processError) return empty(t("无法读取进程"), state.processError, `<button data-action="refresh">${t("重试")}</button>`);
  if (!state.processes) return empty(t("正在读取进程"), t("进程采集独立于窗口绘制运行。"));
  const rows = orderedProcesses(state);
  return `<div class="result-summary"><span>${t("显示")} ${rows.length} ${t("个 · 系统共")} ${state.processes.total} ${t("个")}</span><span>${state.processes.skipped} ${t("个因权限或已退出未读取 ·")} ${timeLabel(state.processes.updatedAt)}</span></div>${rows.length ? `<section class="table-section"><table class="process-table"><thead><tr><th>${t("进程")}</th><th>PID</th><th>CPU</th><th>${t("物理占用")}</th><th>${t("读取")}</th><th>${t("写入")}</th><th>${t("线程")}</th></tr></thead><tbody>${rows.map(p => `<tr><td title="${e(p.path)}"><span class="process-name">${e(p.name)}</span>${p.own ? "" : `<span class="inline-note">${t("系统/其他用户")}</span>`}</td><td class="mono">${p.pid}</td><td class="number">${p.cpuReady ? percent(p.cpu) : "—"}</td><td>${formatMemory(p.memory)}</td><td><span title="${e(p.ioError || "")}">${p.ioReady ? formatRate(p.readRate) : "—"}</span></td><td><span title="${e(p.ioError || "")}">${p.ioReady ? formatRate(p.writeRate) : "—"}</span></td><td>${p.threads}</td></tr>`).join("")}</tbody></table></section>` : empty(t("没有匹配的进程"), t("试试其他名称或 PID，或取消“仅我的进程”。"))}${note((isMac(state) ? t("进程 CPU 以单逻辑核 100% 计，可超过 100%。内存为 phys_footprint；内存共享使进程总和不等于系统已用。仅提供查看，不终止进程。") : t("进程 CPU 以单逻辑核 100% 计，可超过 100%。内存为常驻集 RSS；共享内存会重复计入。无权读取的 I/O 显示为不可用。仅提供查看，不终止进程。")))}`;
}

function connectionView(state: ViewState): string {
  if (state.connectionError) return empty(t("无法读取连接"), state.connectionError, `<button data-action="refresh">${t("重试")}</button>`);
  if (!state.connections) return empty(t("正在读取网络连接"), t("读取当前权限可见的 TCP / UDP 套接字。"));
  const query = state.query.toLowerCase();
  const rows = state.connections.items.filter(c => `${c.process} ${c.pid} ${c.protocol} ${c.local} ${c.remote} ${c.state}`.toLowerCase().includes(query));
  return `${state.connections.warning ? notice(state.connections.warning) : ""}<div class="result-summary"><span>${rows.length} ${t("个连接")}</span><span>${t("快照更新于")} ${timeLabel(state.connections.updatedAt)} ${t("· 点刷新重新读取")}</span></div>${rows.length ? `<section class="table-section"><table class="connection-table"><thead><tr><th>${t("进程 / PID")}</th><th>${t("协议")}</th><th>${t("本地地址")}</th><th>${t("远端地址")}</th><th>${t("状态")}</th></tr></thead><tbody>${rows.map(c => `<tr><td><strong>${e(c.process)}</strong><span class="inline-note">${c.pid}</span></td><td>${e(c.protocol)}<span class="inline-note">${e(c.family)}</span></td><td class="endpoint">${e(c.local)}</td><td class="endpoint">${e(c.remote || "—")}</td><td>${e(c.state || "—")}</td></tr>`).join("")}</tbody></table></section>` : empty(t("没有匹配的连接"), t("当前权限下没有匹配的 TCP / UDP 记录，可调整搜索或刷新。"))}${note((isMac(state) ? t("来源：本机 lsof。只包含当前权限可见的套接字，不提权、不解析远端域名，不向这些地址发起请求。") : t("来源：Windows 系统连接表或 Linux procfs。只包含当前权限可见的套接字；进程归属可能不可见。不提权、不解析远端域名、不发起请求。")))}`;
}

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
  if (state.processError) return empty(t("无法读取 App"), state.processError, `<button data-action="refresh">${t("重试")}</button>`);
  if (!state.processes) return empty(t("正在读取运行中的 App"), t("将同一个 App 包中的进程归并显示。"));
  const apps = groupApps(state.processes.items).filter(app => `${app.name} ${app.path}`.toLowerCase().includes(state.query.toLowerCase()));
  apps.sort((a, b) => state.sort === "name" ? a.name.localeCompare(b.name) : state.sort === "cpu" ? b.cpu - a.cpu : b.memory - a.memory);
  return `<div class="result-summary"><span>${apps.length} ${t("个正在运行的 App")}</span><span>${(isMac(state) ? t("按 App 包路径归并 · 不含未启动的 App") : t("按可执行文件路径归并 · 仅含运行中的进程"))}</span></div>${apps.length ? `<div class="app-list"><div class="app-row app-head"><span>App</span><span>${t("物理占用")}</span><span>CPU</span><span>${t("进程")}</span></div>${apps.map(app => `<button class="app-row" data-app="${e(app.path)}">${appIcon(app.path, isMac(state))}<span class="app-name"><strong>${e(app.name)}</strong><small>${e(app.path)}</small></span><span>${formatMemory(app.memory)}</span><span>${app.cpuReady ? percent(app.cpu) : t("采样中")}</span><span>${app.processes}</span><span class="card-chevron">${icon("chevron")}</span></button>`).join("")}</div>` : empty(t("没有匹配的 App"), t("试试其他名称，或查看完整进程列表。"))}${note((isMac(state) ? t("只读查看运行状态。物理占用来自各进程 phys_footprint 之和；此页面不执行卸载或删除应用数据。") : t("只读查看运行状态。内存为各进程 RSS 之和，共享页可能重复计入；此页面不执行卸载或删除。")))}`;
}

function cleanupView(state: ViewState): string {
  const intro = notice(t("空间分析只读取文件大小，不删除文件。缓存可能仍被 App 使用；下载和废纸篓也不等同于可安全删除空间。"));
  if (state.scanning) return intro + empty(t("正在扫描本机空间"), t("逐项读取配置的目录。遇到权限限制会标注，达到时间或数量上限会返回已读部分。"));
  if (state.scanError) return intro + empty(t("扫描未完成"), state.scanError, `<button data-action="scan">${t("重新扫描")}</button>`);
  if (!state.scan) return intro + `<div class="scan-intro"><span class="scan-icon disk">${icon("cleanup")}</span><h2>${t("先了解空间去了哪里")}</h2><p>${t("分析下列配置目录的文件大小。文件内容不会被读取。")}</p><ul>${(state.info?.scanTargets || []).map(target => `<li><strong>${e(t(target.name))}</strong><span class="path">${e(target.path)}</span></li>`).join("")}</ul><button class="primary-button" data-action="scan">${t("开始空间分析")}</button></div>`;
  return intro + `${!state.scan.complete ? notice(state.scan.warning || t("部分目录无法读取。以下数据仅代表成功读取的文件。")) : ""}<div class="result-summary"><span>${state.scan.complete ? t("扫描完成") : t("扫描结果不完整")}</span><span>${timeLabel(state.scan.finishedAt)} ${t("· 逻辑文件大小，不代表实际可释放空间")}</span></div><div class="scan-groups">${state.scan.groups.map(group => `<section class="scan-group"><div class="scan-group-heading"><div><h2>${e(t(group.name))}</h2><p class="path">${e(group.path)}</p></div><div><strong>${formatCapacity(group.bytes)}</strong><span>${group.files} ${t("个文件")}</span></div></div>${group.error ? notice(group.error) : ""}${group.skipped ? note(`${group.skipped} ${t("项因权限或读取错误跳过。")}`) : ""}<div class="space-list">${group.items.map(item => `<div><span title="${e(item.path)}">${icon("folder")}<strong>${e(item.name)}</strong></span><span>${item.files} ${t("项")}</span><span>${formatCapacity(item.bytes)}</span></div>`).join("") || `<p class="muted">${t("未读取到普通文件")}</p>`}</div></section>`).join("")}</div>${note(t("不跟随符号链接。按第一层目录汇总并显示体积最大的项目；硬链接与 APFS 克隆可能重复计入逻辑大小。扫描范围和资源上限集中配置在 localops.json。"))}`;
}

function aboutView(state: ViewState): string {
  const i = state.info!, m = i.machine;
  return `<div class="about-hero"><span class="machine-icon">${icon("about")}</span><h2>${e(m.hostname)}</h2><p>${e(m.chip)} · ${formatMemory(m.memory)}</p></div>${m.error ? notice(m.error) : ""}<div class="about-grid">${panel(t("这台电脑"), facts([[t("型号标识"), e(m.model)], [t("操作系统"), e(m.os)], [t("架构"), e(m.architecture)], [t("物理 / 逻辑核心"), `${m.physicalCPUs} / ${m.logicalCPUs}`], [t("已运行"), duration(state.snapshot!.uptimeSeconds)]]))}${panel("LocalOps", facts([[t("版本"), e(i.version)], [t("技术"), "Go + MyGo + Vite"], [t("指标刷新"), `${i.refreshSeconds} ${t("秒")}`], [t("磁盘容量刷新"), `${i.diskRefreshSeconds} ${t("秒")}`], [t("进程刷新"), `${i.processRefreshSeconds} ${t("秒")}`], [t("历史保留"), t("最近 1 小时 · 仅本次运行")]]))}</div>${notice(t("支持 macOS、Windows 和 Linux 系统采集。CPU 温度、App 级功耗、卸载、删除及进程终止不在当前功能范围内。"))}${note(t("数据仅在本机读取。GPU、电池和功率计数的可用性取决于硬件、驱动及当前权限；没有计数时显示不可用。"))}`;
}

export function renderBody(state: ViewState): string {
  if (state.page === "settings") return "";
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
