import { expect, test } from "bun:test";
import { sensorsView } from "../src/sensors-view";
import { actionControls, renderBody, type ViewState } from "../src/views";
import type { Snapshot } from "../src/mygo";
import { t } from "../src/i18n";
import { temperatureColorThresholds } from "../rockshinops.json";

const snapshot = {
  sensors: { temperatureSummary: { cpu: 55, gpu: 48, ssd: 34, battery: 32 }, temperatures: [{key:"Tp01",component:"cpu",kind:"performance",celsius:55,source:"AppleSMC"}], electrical: [{key:"VD0R",kind:"voltage",value:28,source:"AppleSMC"}, {key:"ID0R",kind:"current",value:-0.2,source:"AppleSMC"}], dieTemperature: 43.9, temperatureSource: "HID die maximum", fans: [{ index: 0, rpm: 0, targetRPM: 1350, minRPM: null, maxRPM: 5777, source: "SMC F0Ac" }], powerRails: [{ key: "PSTR", value: 0, source: "AppleSMC" }] },
  thermal: { pressure: "nominal", source: "macOS", error: "" },
  battery: { present: false, error: "" },
  powerAssertions: { items: [], error: "", source: "IOPM" },
} as Snapshot;

function withIcons(run: () => void) {
  const previous = globalThis.document;
  globalThis.document = { createElementNS: () => ({ setAttribute() {}, appendChild() {}, outerHTML: "<svg></svg>" }) } as unknown as Document;
  try { run(); } finally { globalThis.document = previous; }
}

test("sensor readings preserve measured zero and never infer a fanless device from missing readings", () => withIcons(() => {
  const html = sensorsView(snapshot, "darwin", [], 60, 2);
  expect(html).toContain("43.9");
  expect(html).toContain("55.0");
  expect(html).toContain("1350");
  expect(html).toContain("5777");
  expect(html).not.toContain('data-sensor-detail="power"');
  expect(html).not.toContain("PSTR");
  expect(html).not.toContain("VD0R");
  expect(html).toContain(t("等待两个有效样本"));
  expect(html).toContain('<strong>0</strong><span class="sensor-unit">rpm</span>');
  const missing = sensorsView({ ...snapshot, sensors: { temperatureSummary: {cpu:null,gpu:null,ssd:null,battery:null}, temperatures: [], electrical: [], dieTemperature: null, temperatureSource: "", fans: [], powerRails: [] } }, "darwin", [], 60, 2);
  expect(missing).toContain(t("未获取到风扇读数；设备可能无风扇，或未提供此数据。"));
  expect(missing).not.toContain("<strong>0");
}));

test("sensor availability and thermal errors remain explicit without implying a nominal state", () => withIcons(() => {
  const html = sensorsView({ ...snapshot, thermal: { ...snapshot.thermal, error: '<source unavailable>' } }, "darwin", [], 60, 2);
  expect(html).toContain("&lt;source unavailable&gt;");
  expect(html).not.toContain(t("热压力正常"));
  for (const platform of ["windows", "linux"]) {
    const unsupported = sensorsView(snapshot, platform, [], 60, 2);
    expect(unsupported).toContain(t("当前平台暂不支持传感器读取"));
    expect(unsupported).not.toContain("43.9");
  }
}));

test("Energy and Sensors have separate content and Sensors offers pause with real temperature history", () => withIcons(() => {
  const state = { history: [], range: 60, page: "energy", info: { machine: { platform: "darwin" }, powerAssertionRefreshSeconds: 5 }, snapshot } as ViewState;
  const energy = renderBody(state);
  expect(energy).toContain(t("睡眠与唤醒"));
  expect(energy).not.toContain(t("芯片温度"));
  expect(energy).not.toContain(t("功率通道"));
  const sensors = renderBody({ ...state, page: "sensors" });
  expect(sensors).toContain(t("芯片温度"));
  expect(sensors).not.toContain(t("睡眠与唤醒"));
  const actions = actionControls({ ...state, page: "sensors" });
  expect(actions).toContain('data-action="pause"');
  expect(actions).toContain('data-range="60"');
}));


test("temperature color bands follow displayed values without changing native thermal pressure", () => withIcons(() => {
  const { mediumCelsius, highCelsius } = temperatureColorThresholds;
  expect(mediumCelsius).toBeLessThan(highCelsius);
  const cases: [number | null, string][] = [[0, "low"], [mediumCelsius - .1, "low"], [mediumCelsius, "medium"], [highCelsius - .1, "medium"], [highCelsius - .04, "high"], [highCelsius, "high"], [null, "unavailable"]];
  for (const [value, level] of cases) {
    const html = sensorsView({ ...snapshot, sensors: { ...snapshot.sensors, temperatureSummary: { ...snapshot.sensors.temperatureSummary, cpu: value }, temperatures: [{ ...snapshot.sensors.temperatures[0]!, celsius: value }], dieTemperature: value } }, "darwin", [], 60, 2);
    expect(html).toContain(`data-row-key="summary-cpu" data-temperature-level="${level}"`);
    expect(html.match(new RegExp(`<dd data-temperature-level="${level}">`, "g"))?.length).toBe(2);
    expect(html).toContain(t("热压力正常"));
    expect(html).not.toContain('sensor-pressure warning');
  }
}));
