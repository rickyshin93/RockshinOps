import type { ViewState } from "./views";
import { t } from "./i18n";
import { escapeHTML as e } from "./format";
import { icon, processAppIcon } from "./icons";
import { filterPorts, listeningPorts, type ListeningPort } from "./ports";

function scopeBadge(row: ListeningPort): string {
  if (row.scopes.has("wildcard")) return `<span class="port-scope public" title="${e(t("绑定所有接口，局域网可能可达"))}">${t("所有接口")}</span>`;
  if (row.scopes.has("interface")) return `<span class="port-scope public" title="${e(t("绑定指定网卡地址"))}">${t("指定接口")}</span>`;
  if (row.scopes.has("loopback")) return `<span class="port-scope loopback">${t("本机回环")}</span>`;
  return "";
}

export function portsView(state: ViewState): string {
  const all = listeningPorts(state.connections?.items || [], state.processes?.items || []);
  const rows = filterPorts(all, state.query, state.portScope || "all");
  const favorites = state.favoritePorts || [];
  const error = state.connectionError;
  const message = state.portMessage;
  const loading = !state.connections;
  const action = state.portAction;
  const platform = state.info?.machine.platform || "";
  const favorite = (port: number) => `<button class="icon-button port-star ${favorites.includes(port) ? "selected" : ""}" data-favorite-port="${port}" aria-pressed="${favorites.includes(port)}" aria-label="${e(t(favorites.includes(port) ? "取消收藏端口 {{port}}" : "收藏端口 {{port}}", { port }))}" title="${e(t(favorites.includes(port) ? "取消收藏端口 {{port}}" : "收藏端口 {{port}}", { port }))}" ${state.portsSaving || !state.portsLoaded ? "disabled" : ""}>${icon("star")}</button>`;
  const closePort = (row: ListeningPort) => {
    const label = t("关闭端口 {{port}}", { port: row.port });
    if (loading || error) return `<button class="icon-button port-close" disabled aria-label="${e(label)}" title="${e(t("连接数据暂不可用"))}">${icon("power")}</button>`;
    if (!row.identity || row.pid <= 1 || !row.process) return `<button class="icon-button port-close" disabled aria-label="${e(label)}" title="${e(t("进程信息不可用，无法结束"))}">${icon("power")}</button>`;
    const busy = action?.pid === row.pid && action.busy;
    return `<button class="icon-button port-close" data-close-port="${row.pid}" aria-label="${e(label)}" title="${e(label)}" ${busy || action?.busy ? "disabled" : ""} aria-busy="${busy}">${busy ? icon("refresh") : icon("power")}</button>`;
  };
  const portCount = new Set(all.map(row => row.port)).size;
  const loopbackCount = new Set(all.filter(row => row.scopes.has("loopback") && !row.scopes.has("wildcard") && !row.scopes.has("interface")).map(row => row.port)).size;
  const publicCount = new Set(all.filter(row => row.scopes.has("wildcard") || row.scopes.has("interface")).map(row => row.port)).size;
  return `<div class="ports-panel">
    <div class="ports-toolbar"><div class="segmented" role="group" aria-label="${t("监听范围")}">${([["all", "全部"], ["loopback", "本机回环"], ["wildcard", "所有接口"]] as const).map(([scope, label]) => `<button data-port-scope="${scope}" aria-pressed="${(state.portScope || "all") === scope}" class="${(state.portScope || "all") === scope ? "selected" : ""}">${t(label)}</button>`).join("")}</div><div class="ports-summary"><span class="ports-stat"><strong>${loading ? "—" : portCount}</strong> ${t("TCP 端口")}</span><span class="ports-stat"><strong>${loading ? "—" : loopbackCount}</strong> ${t("本机回环")}</span><span class="ports-stat"><strong>${loading ? "—" : publicCount}</strong> ${t("对外绑定")}</span></div></div>
    <div class="ports-scroll"><div class="ports-favorites"><span class="ports-favorites-label">${icon("star")}${t("收藏")}</span><div class="ports-favorites-list">${favorites.length ? favorites.map(port => `<span class="port-favorite"><button data-port-query="${port}" aria-label="${e(t("筛选端口 {{port}}", { port }))}" aria-pressed="${state.query === `:${port}`}"><strong>${port}</strong><span>${t(loading || error ? "状态未知" : all.some(row => row.port === port) ? "监听中" : "未发现监听")}</span></button>${favorite(port)}</span>`).join("") : `<span class="ports-hint">${t("点击端口旁的星标收藏，重启后仍保留。")}</span>`}</div></div>
    ${state.portsError ? `<div class="notice error" role="alert">${e(state.portsError)} <button data-action="load-port-favorites">${t("重试")}</button></div>` : ""}
    ${message ? `<p class="port-message ${message.error ? "error" : ""}" role="${message.error ? "alert" : "status"}">${e(message.text)}</p>` : ""}
    ${error ? `<div class="notice error" role="alert">${e(t(state.connections ? "更新中断 · 上次数据" : "无法读取连接"))} · ${e(error)} <button data-action="refresh">${t("重试")}</button></div>` : ""}
    ${state.connections?.warning ? `<div class="notice">${e(state.connections.warning)}</div>` : ""}
    ${loading ? `<div class="empty-state"><h2>${t(error ? "无法读取连接" : "正在读取网络连接")}</h2><p>${t("读取当前权限可见的 TCP 监听套接字。")}</p></div>` : rows.length ? `<table class="ports-table" aria-label="${t("开发端口")}"><thead><tr><th>${t("端口")}</th><th>${t("占用进程")}</th><th>${t("监听地址")}</th><th>App / ${t("监听范围")}</th><th aria-label="${t("操作")}"></th></tr></thead><tbody>${rows.map(row => `<tr data-row-key="port-${row.port}-${row.pid}"><td><div class="port-number">${favorite(row.port)}<strong>${row.port}</strong></div></td><td><div class="port-owner">${processAppIcon(row.app, platform === "darwin")}<div class="port-owner-copy"><button class="port-process" data-port-process="${row.pid}"><span title="${e(row.process || t("进程不可用"))}">${e(row.process || t("进程不可用"))}</span><small>PID ${row.pid}</small></button></div></div></td><td><div class="port-addresses">${row.addresses.map(address => `<div class="port-address"><code title="${e(address)}">${e(address)}</code><button class="icon-button" data-copy-port="${e(address)}" aria-label="${e(t("复制监听地址 {{address}}", { address }))}" title="${e(t("复制监听地址 {{address}}", { address }))}">${icon("copy")}</button></div>`).join("")}</div></td><td><div class="port-meta">${row.app ? `<span class="port-app" title="${e(row.app)}">${e(row.app.split(/[\\/]/).at(-1) || row.app)}</span>` : ""}${scopeBadge(row)}</div></td><td><div class="port-actions"><button class="port-details" data-port-process="${row.pid}" aria-label="${e(t("查看进程 {{pid}}", { pid: row.pid }))}">${t("详情")}${icon("chevron")}</button>${platform === "linux" ? "" : closePort(row)}</div></td></tr>`).join("")}</tbody></table>` : `<div class="empty-state">${icon("ports")}<h2>${t(state.query || state.portScope !== "all" && state.portScope ? "没有匹配的监听端口" : "暂未发现 TCP 监听端口")}</h2><p>${t("仅显示当前权限可见的监听记录；未发现不代表端口可用。")}</p>${state.query || state.portScope && state.portScope !== "all" ? `<button data-action="clear-port-filters">${t("清除筛选")}</button>` : `<button data-action="refresh">${t("立即刷新")}</button>`}</div>`}</div>
    <footer class="ports-footer"><span>${t("{{count}} 条监听记录", { count: rows.length })}</span><span>${t("仅 TCP LISTEN · 关闭端口会结束其占用进程。")}</span></footer>
  </div>`;
}
