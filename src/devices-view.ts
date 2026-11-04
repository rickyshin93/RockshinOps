import type { DeviceState, WiFiNetwork } from "./mygo";
import { t } from "./i18n";
import { escapeHTML as e, timeLabel } from "./format";
import { icon } from "./icons";
import { wifiSignalThresholds } from "../rockshinops.json";

export interface DevicesViewState {
  devices?: DeviceState | null;
  deviceBusy?: string;
  deviceError?: string;
  deviceReadError?: string;
  deviceMessage?: string;
  deviceNetworks?: WiFiNetwork[] | null;
  deviceScanAt?: string;
  deviceScanError?: string;
  deviceVolumeDraft?: number | null;
  deviceVolumeTarget?: string | null;
}
const missing = "—";
const reading = (value: number | null | undefined, unit: string, digits = 0) => value == null ? missing : `${value.toFixed(digits)} ${unit}`;
function signalLevel(value: number | null): number {
  if (value == null) return 0;
  if (value >= wifiSignalThresholds.strongDbm) return 3;
  if (value >= wifiSignalThresholds.fairDbm) return 2;
  return 1;
}
function signalLabel(value: number | null): string {
  return t(["信号未知", "信号偏弱", "信号一般", "信号强"][signalLevel(value)]!);
}
function signalIcon(value: number | null): string {
  const level = signalLevel(value);
  return `<svg class="device-signal" data-level="${level}" viewBox="0 0 24 24" aria-hidden="true">${[0, 1, 2].map(i => `<rect x="${3 + i * 7}" y="${16 - i * 5}" width="4" height="${6 + i * 5}" rx="1.3" class="${i < level ? "active" : ""}"/>`).join("")}</svg>`;
}
function button(label: string, action: string, busy: string, id = "", disabled = false): string {
  return `<button class="device-button" data-device-action="${action}" data-device-id="${e(id)}" ${busy || disabled ? "disabled" : ""}>${e(t(busy === `${action}:${id}` ? "处理中…" : label))}</button>`;
}
function settings(section: string): string {
  return `<button class="device-settings" data-device-settings="${section}">${e(t("系统设置"))}${icon("chevron")}</button>`;
}
function permission(kind: "location" | "bluetooth", status: string, busy: string): string {
  if (status === "allowed") return "";
  const location = kind === "location";
  const detail = location
    ? status === "notDetermined" ? "显示 Wi-Fi 名称和扫描附近网络需要定位权限。" : "定位权限未开放；请在系统设置中允许 RockshinOps 访问定位服务。"
    : status === "notDetermined" ? "允许蓝牙访问后，可查看和连接已配对设备。" : "蓝牙访问未开放；请在系统设置中允许 RockshinOps 访问蓝牙。";
  return `<div class="device-permission"><p>${e(t(detail))}</p>${status === "notDetermined" ? button(location ? "显示网络名称" : "允许蓝牙访问", location ? "location" : "bluetoothPermission", busy) : settings(location ? "location" : "bluetoothPrivacy")}</div>`;
}
function summary(iconName: string, label: string, title: string, detail: string, band = ""): string {
  return `<div class="device-summary-item">${icon(iconName)}<div><span>${e(t(label))}</span><strong>${e(title)}${band ? `<span class="device-wifi-band">${e(band)}</span>` : ""}</strong><small>${e(detail)}</small></div></div>`;
}
function pair(label: string, value: string): string {
  return `<div><dt>${e(t(label))}</dt><dd>${e(value)}</dd></div>`;
}
export function devicesView(state: DevicesViewState): string {
  const data = state.devices;
  const busy = state.deviceBusy || "";
  const error = state.deviceError || state.deviceReadError || data?.error;
  const feedback = `${error ? `<div class="device-error" role="alert">${e(t("设备操作或读取失败，请重试。"))} ${e(error)}</div>` : ""}${state.deviceMessage ? `<p class="device-message" role="status">${e(state.deviceMessage)}</p>` : ""}`;
  if (!data) return `<div class="devices-page">${feedback}<div class="empty-state"><h2>${e(t(busy ? "正在读取设备…" : "等待设备数据"))}</h2><p>${e(t("点击刷新读取本机设备。"))}</p></div></div>`;
  if (data.error) return `<div class="devices-page">${feedback}</div>`;
  if (!data.supported) return `<div class="empty-state"><h2>${e(t("当前平台暂不支持设备连接"))}</h2><p>${e(t("Wi-Fi、蓝牙和声音控制目前仅在 macOS 上可用。"))}</p></div>`;
  const { wifi, bluetooth, audio } = data;
  const connected = bluetooth.devices.filter(device => device.connected);
  const output = audio.outputs.find(device => device.id === audio.defaultId);
  const wifiTitle = !wifi.available ? t("未检测到 Wi-Fi 接口") : !wifi.powered ? t("Wi-Fi 已关闭") : wifi.connected ? wifi.ssid || t("Wi-Fi 已连接") : t("Wi-Fi 未连接");
  const btTitle = bluetooth.authorization !== "allowed" ? t("需要蓝牙权限") : !bluetooth.available ? t("未检测到蓝牙控制器") : !bluetooth.powered ? t("蓝牙已关闭") : t("{{count}} 台已连接", { count: connected.length });
  const volume = state.deviceVolumeDraft ?? audio.volume;
  const wifiConnected = wifi.available && wifi.powered && wifi.connected;
  const signal = wifiConnected ? signalLabel(wifi.signal) : wifiTitle;
  const wifiHint = !wifi.available ? "这台电脑没有可用的 Wi-Fi。"
    : !wifi.powered ? "请在系统设置中打开 Wi-Fi。"
    : !wifi.connected ? "请在系统设置中选择要连接的网络。"
    : wifi.signal == null ? "已连接 Wi-Fi，但系统暂未提供信号强度。"
    : wifi.signal < wifiSignalThresholds.fairDbm ? "可以尝试靠近路由器，或减少中间的遮挡。"
    : "信号强弱只反映电脑与路由器的无线连接，不代表上网速度。";
  const wifiDetails = [
    pair("信号", reading(wifi.signal, "dBm")), pair("背景噪声", reading(wifi.noise, "dBm")),
    pair("链路速率", reading(wifi.rate, "Mbps")), pair("信道", [wifi.channel, wifi.band, wifi.width == null ? "" : `${wifi.width} MHz`].filter(v => v != null && v !== "").join(" · ") || missing),
    pair("无线标准", wifi.phy || missing), pair("加密方式", wifi.security || missing),
    pair("网卡接口", wifi.interface || missing), pair("信噪比", wifi.signal == null || wifi.noise == null ? missing : reading(wifi.signal - wifi.noise, "dB")),
  ].join("");
  const networks = state.deviceNetworks;
  const scan = `<details class="device-disclosure device-scan" data-row-key="wifi-scan"><summary>${icon("chevron")}<span>${e(t("附近的 Wi-Fi"))}</span><small>${networks == null ? e(t("点击查看")) : e(t("{{count}} 个网络", { count: networks.length }))}</small></summary><div class="device-scan-body"><div class="device-scan-toolbar"><p>${e(t("这里只查看附近网络。要连接或更换 Wi-Fi，请打开系统设置。"))}</p>${button("扫描网络", "scan", busy, "", !wifi.powered || wifi.location !== "allowed")}</div>${state.deviceScanError ? `<p class="device-error" role="alert">${e(state.deviceScanError)}</p>` : ""}${networks == null ? "" : networks.length ? `<ul class="device-network-list">${networks.map(network => `<li><strong>${e(network.ssid)}</strong><span class="device-network-signal">${signalIcon(network.signal)}<span>${e([signalLabel(network.signal), t(network.secure ? "已加密" : "未加密")].join(" · "))}</span></span></li>`).join("")}</ul>` : `<p class="device-empty">${e(t("未扫描到可显示名称的网络。"))}</p>`}${state.deviceScanAt ? `<p class="device-footnote">${e(t("上次扫描"))} · ${e(timeLabel(state.deviceScanAt))}</p>` : ""}</div></details>`;
  const bluetoothRows = bluetooth.devices.map(device => `<li data-row-key="${e(device.id)}"><span class="device-row-icon">${icon(device.kind === "audio" ? "headphones" : device.kind === "computer" ? "about" : device.kind === "peripheral" ? "mouse" : "bluetooth")}</span><div class="device-row-copy"><strong>${e(device.name)}</strong><small>${e(t(device.connected ? "已连接" : "未连接"))}${device.battery == null ? "" : ` · ${e(t("电池"))} ${device.battery}%`}</small></div>${button(device.connected ? "断开" : "连接设备", device.connected ? "disconnect" : "connect", busy, device.id, !bluetooth.powered)}</li>`).join("");
  const audioRows = audio.outputs.map(device => `<li data-row-key="${e(device.id)}"><button class="device-output ${device.id === audio.defaultId ? "selected" : ""}" data-device-action="output" data-device-id="${e(device.id)}" aria-pressed="${device.id === audio.defaultId}" ${busy || device.id === audio.defaultId ? "disabled" : ""}>${icon("speaker")}<span>${e(device.name)}</span>${device.id === audio.defaultId ? `<span class="device-output-current">${e(t("当前"))}${icon("check")}</span>` : ""}</button></li>`).join("");
  return `<div class="devices-page">${feedback}<div class="device-summary" aria-label="${e(t("设备概况"))}">${summary("wifi", "Wi-Fi", wifiTitle, wifiConnected ? signal : t("无线网络"), wifiConnected ? wifi.band : "")}${summary("bluetooth", "蓝牙", btTitle, connected.map(device => device.name).join("、") || t("已配对的设备"))}${summary("speaker", "声音播放设备", output?.name || t("无输出设备"), audio.volume == null ? t("音量未知") : t("音量 {{value}}%", { value: Math.round(audio.volume) }))}</div>
  <section class="device-panel device-wifi-panel"><header><div class="device-wifi-title"><h2>${icon("wifi")}Wi-Fi</h2><span class="device-wifi-status" title="${e(t(wifiHint))}">${signalIcon(wifiConnected ? wifi.signal : null)}<strong>${e(signal)}</strong></span></div>${settings("wifi")}</header>${permission("location", wifi.location, busy)}${wifi.connected && !wifi.ssid && wifi.location === "allowed" ? `<p class="device-empty">${e(t("系统未提供当前网络名称。"))}</p>` : ""}${scan}<details class="device-disclosure device-technical" data-row-key="wifi-technical" open><summary>${icon("chevron")}<span>${e(t("技术详情"))}</span><small>${e(t("信号、速率与信道"))}</small></summary><dl class="device-wifi-facts">${wifiDetails}</dl><p class="device-footnote">${e(t(wifiHint))}</p><p class="device-footnote">${e(t("链路速率是电脑与路由器商定的连接速率，不是实际下载速度。"))}</p><p class="device-footnote">${e(t("信号分级参考：强 ≥ {{strong}} dBm；一般 ≥ {{fair}} dBm；更低为偏弱。", { strong: wifiSignalThresholds.strongDbm, fair: wifiSignalThresholds.fairDbm }))} · ${e(wifi.source)}</p></details></section>
  <div class="device-columns"><section class="device-panel"><header><h2>${icon("bluetooth")}${e(t("蓝牙设备"))}</h2>${settings("bluetooth")}</header>${permission("bluetooth", bluetooth.authorization, busy)}${bluetooth.authorization === "allowed" ? bluetoothRows ? `<ul class="device-list">${bluetoothRows}</ul>` : `<p class="device-empty">${e(t(!bluetooth.available ? "未检测到蓝牙控制器" : bluetooth.powered ? "暂无已配对设备。请在系统设置中配对。" : "蓝牙已关闭，请在系统设置中开启。"))}</p>` : ""}<p class="device-footnote">${e(t("新设备请先在系统设置中配对。没有显示电量，表示系统暂未提供。"))} · ${e(bluetooth.source)}</p></section>
  <section class="device-panel"><header><h2>${icon("speaker")}${e(t("声音播放设备"))}</h2>${settings("audio")}</header>${audio.error ? `<p class="device-error" role="alert">${e(audio.error)}</p>` : ""}${audioRows ? `<ul class="device-list device-outputs">${audioRows}</ul>` : `<p class="device-empty">${e(t("未检测到声音输出设备。"))}</p>`}<div class="device-volume"><div><label for="device-volume">${e(t("音量"))}</label><output for="device-volume" id="device-volume-value">${reading(volume, "%")}</output></div><input type="range" id="device-volume" min="0" max="100" step="1" value="${volume == null ? 0 : Math.round(volume)}" data-device-volume="${e(state.deviceVolumeTarget ?? audio.defaultId)}" ${(busy && !busy.startsWith("volume:")) || !audio.volumeSettable || audio.volume == null ? "disabled" : ""} aria-valuetext="${e(reading(volume, "%"))}">${!audio.volumeSettable ? `<p class="device-footnote">${e(t("请用设备上的按键或旋钮调节音量。"))}</p>` : ""}</div><p class="device-footnote">${e(t("设为默认播放设备。部分应用可单独选择。"))} · ${e(audio.source)}</p></section></div></div>`;
}
