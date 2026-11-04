import type { Snapshot, TemperatureReading, Trend } from "./mygo";
import { escapeHTML as e } from "./format";
import { icon } from "./icons";
import { t } from "./i18n";
import { lineChart, type TrendKey } from "./charts";
import { temperatureColorThresholds } from "../rockshinops.json";

const thermalLabels: Record<string, string> = {
  nominal: "热压力正常", moderate: "热压力中等", heavy: "热压力较高",
  trapping: "热压力严重", sleeping: "过热即将睡眠",
};
const kindLabels: Record<string, string> = {
  efficiency: "CPU 能效通道", performance: "CPU 性能通道", cpu: "CPU 温度通道", gpu: "GPU 温度通道",
  nand: "SSD / NAND", storage: "存储", battery: "电池", airflowLeft: "左侧风道", airflowRight: "右侧风道", wireless: "无线模块",
};
const summaries = [
  ["cpu", "CPU", "cpuTemperature"], ["gpu", "GPU", "gpuTemperature"],
  ["ssd", "SSD / NAND", "ssdTemperature"], ["battery", "电池", "batteryTemperature"],
] as const;

function reading(value: number | null, unit: string, digits = 0): string {
  return value == null ? `<span class="sensor-missing">${t("暂无读数")}</span>` : `<strong>${value.toFixed(digits)}</strong><span class="sensor-unit">${unit}</span>`;
}
function temperatureLevel(value: number | null): string {
  if (value == null) return "unavailable";
  const displayed = Number(value.toFixed(1));
  return displayed >= temperatureColorThresholds.highCelsius ? "high" : displayed >= temperatureColorThresholds.mediumCelsius ? "medium" : "low";
}
function temperatureRows(readings: TemperatureReading[]): string {
  return `<dl class="sensor-channel-list">${readings.map(r => `<div data-row-key="temperature-${e(r.key)}"><dt><span>${e(t(kindLabels[r.kind] || r.kind))}</span><small>${e(r.key)}</small></dt><dd data-temperature-level="${temperatureLevel(r.celsius)}">${reading(r.celsius, "°C", 1)}</dd></div>`).join("")}</dl>`;
}

export function sensorsView(snapshot: Snapshot, platform: string, history: Trend[], range: number, interval: number): string {
  if (platform !== "darwin") return `<div class="empty-state"><h2>${t("当前平台暂不支持传感器读取")}</h2><p>${t("芯片温度与风扇目前仅支持部分 Mac 机型。")}</p></div>`;
  const { sensors: s, thermal } = snapshot;
  const pressure = thermal.error ? t("热压力不可用") : t(thermalLabels[thermal.pressure] || thermal.pressure);
  const elevated = !thermal.error && thermal.pressure !== "nominal";
  const end = Date.parse(snapshot.updatedAt);
  const temperatureGroup = (component: string, name: string) => {
    const rows = s.temperatures.filter(r => r.component === component);
    if (!rows.length) return "";
    return `<details class="sensor-channel-group" data-row-key="group-${component}" data-sensor-detail="temperature-${component}"><summary><span>${name}</span><span>${e(t("{{count}} 个通道", { count: rows.length }))}</span>${icon("chevron")}</summary>${temperatureRows(rows)}</details>`;
  };
  const supporting = ["ssd", "storage", "battery", "cooling"].flatMap(component => s.temperatures.filter(r => r.component === component));
  const fans = s.fans.length ? s.fans.map(f => `<div class="sensor-fan" data-row-key="fan-${f.index}"><div class="sensor-fan-heading"><h3>${icon("fan")}${t("风扇")} ${f.index + 1}</h3><div>${reading(f.rpm, "rpm")}</div></div><dl class="sensor-fan-specs">${([["目标转速", f.targetRPM], ["最低转速", f.minRPM], ["最高转速", f.maxRPM]] as const).map(([name, value]) => `<div><dt>${t(name)}</dt><dd>${reading(value, "rpm")}</dd></div>`).join("")}</dl></div>`).join("") : `<p class="sensor-empty">${t("未获取到风扇读数；设备可能无风扇，或未提供此数据。")}</p>`;
  return `<div class="sensors-page">
    <section class="sensor-surface" aria-labelledby="sensor-temperature-title">
      <div class="sensor-section-heading"><h2 id="sensor-temperature-title"><span class="content-icon">${icon("sensors")}</span>${t("温度趋势")}</h2><span class="sensor-pressure${thermal.error ? " unavailable" : elevated ? " warning" : ""}">${e(pressure)}</span></div>
      ${thermal.error ? `<p class="sensor-error" role="status">${e(thermal.error)}</p>` : ""}
      <p class="sensor-hint sensor-summary-note">${t("各类有效通道的最高读数；曲线空缺表示没有有效样本。")}</p>
      <div class="sensor-temperature-overview">${summaries.map(([component, name, key]) => `<div class="sensor-temperature-stat" data-row-key="summary-${component}" data-temperature-level="${temperatureLevel(s.temperatureSummary[component])}"><h3>${t(name)}</h3><div class="sensor-reading">${reading(s.temperatureSummary[component], "°C", 1)}</div><div class="sensor-temperature-chart">${lineChart(history, key as TrendKey, end, range, `${t(name)} · °C`, 125, undefined, interval)}</div></div>`).join("")}</div>
      <div class="sensor-chart-footer"><div class="sensor-temperature-legend" aria-label="${t("温度颜色区间")}" title="${t("温度颜色仅区分读数高低，不代表硬件告警或降频。")}"><span data-temperature-level="low"><i aria-hidden="true"></i>&lt; ${temperatureColorThresholds.mediumCelsius} °C</span><span data-temperature-level="medium"><i aria-hidden="true"></i>${temperatureColorThresholds.mediumCelsius}–&lt;${temperatureColorThresholds.highCelsius} °C</span><span data-temperature-level="high"><i aria-hidden="true"></i>≥ ${temperatureColorThresholds.highCelsius} °C</span></div><p class="sensor-chart-scale">${t("温度刻度：0–125 °C")}</p></div>
    </section>
    <div class="sensor-details-grid">
      <section class="sensor-surface" aria-labelledby="sensor-readings-title"><div class="sensor-section-heading"><h2 id="sensor-readings-title">${t("温度读数")}</h2></div><p class="sensor-hint sensor-summary-note">${t("CPU / GPU 通道不对应单个物理核心。")}</p>
        ${temperatureGroup("cpu", "CPU")}${temperatureGroup("gpu", "GPU")}${temperatureRows(supporting)}
        ${!s.temperatures.length ? `<p class="sensor-empty">${t("当前设备未提供可用的组件温度。")}</p>` : ""}
        <dl class="sensor-channel-list sensor-die"><div><dt><span>${t("芯片温度")}</span><small>${t("HID 最高读数")}</small></dt><dd data-temperature-level="${temperatureLevel(s.dieTemperature)}">${reading(s.dieTemperature, "°C", 1)}</dd></div></dl>
      </section>
      <section class="sensor-surface" aria-labelledby="sensor-fans-title"><div class="sensor-section-heading"><h2 id="sensor-fans-title">${t("风扇")}</h2></div><p class="sensor-hint sensor-summary-note">${t("只读监测，转速由系统控制。")}</p>${fans}</section>
    </div>
    <details class="sensor-disclosure sensor-method" data-sensor-detail="sources">
      <summary><span class="sensor-disclosure-title">${t("读数来源与说明")}</span>${icon("chevron")}</summary>
      <div class="sensor-disclosure-body"><p>${t("组件温度由 AppleSMC 已知机型映射读取，未映射或无效读数显示为不可用。")}</p><p>${t("热压力是 macOS 报告的系统热状态；它不是 CPU 温度或实际降频测量。")}</p><dl class="sensor-sources">${[[t("芯片温度"), s.temperatureSource || t("暂无读数")], [t("热压力"), thermal.source || t("暂无读数")], ...s.fans.map(f => [`${t("风扇")} ${f.index + 1}`, f.source])].map(([name, source]) => `<div><dt>${e(name!)}</dt><dd>${e(source!)}</dd></div>`).join("")}</dl></div>
    </details>
  </div>`;
}
