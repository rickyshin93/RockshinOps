import { expect, test } from "bun:test";
import { devicesView } from "../src/devices-view";
import type { DeviceState } from "../src/mygo";
import { t } from "../src/i18n";
import { wifiSignalThresholds } from "../rockshinops.json";

const data: DeviceState = {
  supported: true, updatedAt: "2026-11-04T03:00:00Z", error: "",
  wifi: { available: true, powered: true, connected: true, interface: "en0", ssid: "", location: "notDetermined", signal: -55, noise: -100, rate: 286, channel: 52, band: "5 GHz", width: 20, phy: "Wi-Fi 6", security: "WPA2 Enterprise", source: "CoreWLAN" },
  bluetooth: { available: true, powered: true, authorization: "allowed", source: "IOBluetooth", devices: [{id:"paired-1",name:"<unsafe device>",kind:"audio",connected:true,battery:0}] },
  audio: { outputs: [{id:"speaker",name:"Mac speakers"}],defaultId:"speaker",volume:0,volumeSettable:true,error:"",source:"CoreAudio" },
};
function withIcons(run: () => void) {
  const previous = globalThis.document;
  globalThis.document = { createElementNS: () => ({ setAttribute() {}, appendChild() {}, outerHTML: "<svg></svg>" }) } as unknown as Document;
  try { run(); } finally { globalThis.document = previous; }
}
test("redacted SSID does not imply disconnection; genuine zero volume and battery remain visible", () => withIcons(() => {
  const html = devicesView({ devices: data });
  expect(html).toContain(t("Wi-Fi 已连接"));
  expect(html).not.toContain(t("Wi-Fi 未连接"));
  expect(html).toContain('data-device-action="location"');
  expect(html).toContain('value="0"');
  expect(html).toContain(`${t("电池")} 0%`);
  expect(html).toContain("&lt;unsafe device&gt;");
  expect(html).not.toContain("<unsafe device>");
}));
test("permission denial routes to settings, and unsupported software volume cannot be changed", () => withIcons(() => {
  const html = devicesView({devices:{...data,wifi:{...data.wifi,location:"denied"},bluetooth:{...data.bluetooth,authorization:"denied",devices:[]},audio:{...data.audio,volume:null,volumeSettable:false}}});
  expect(html).toContain('data-device-settings="location"');
  expect(html).toContain('data-device-settings="bluetoothPrivacy"');
  expect(html).not.toContain('data-device-action="location"');
  expect(html).toMatch(/data-device-volume="speaker" disabled/);
  expect(html).not.toContain('data-device-action="connect"');
}));
test("audio controls retain output identity and volume drafts; scanned names are escaped", () => withIcons(() => {
  const html = devicesView({devices:data,deviceNetworks:[{ssid:"Test <Network>",signal:-55,band:"5 GHz",channel:52,secure:true}],deviceVolumeDraft:25});
  expect(html).toContain('data-device-volume="speaker"');
  expect(html).toContain('value="25"');
  expect(html).toContain('Test &lt;Network&gt;');
}));
test("native failures and unsupported platforms do not show plausible empty readings", () => withIcons(() => {
  const failed = devicesView({devices:{...data,error:"read failed"}});
  expect(failed).toContain("read failed"); expect(failed).not.toContain("286");
  const unsupported = devicesView({devices:{...data,supported:false}});
  expect(unsupported).toContain(t("当前平台暂不支持设备连接")); expect(unsupported).not.toContain('data-device-action=');
}));
test("Wi-Fi signal descriptions respect configured boundaries and unknown readings", () => withIcons(() => {
  const { strongDbm, fairDbm } = wifiSignalThresholds;
  expect(strongDbm).toBeGreaterThan(fairDbm);
  expect(strongDbm).toBeLessThan(0);
  const cases: [number | null, string][] = [[strongDbm, "信号强"], [strongDbm - 1, "信号一般"], [fairDbm, "信号一般"], [fairDbm - 1, "信号偏弱"], [null, "信号未知"]];
  for (const [signal, label] of cases) {
    const html = devicesView({ devices: { ...data, wifi: { ...data.wifi, signal } } });
    expect(html).toContain(`<strong>${t(label)}</strong>`);
    expect(html).toContain(`data-level="${signal == null ? 0 : signal >= strongDbm ? 3 : signal >= fairDbm ? 2 : 1}"`);
  }
}));
test("off or disconnected Wi-Fi does not present an old signal as current", () => withIcons(() => {
  for (const wifi of [{ ...data.wifi, connected: false }, { ...data.wifi, powered: false }, { ...data.wifi, available: false }]) {
    const html = devicesView({ devices: { ...data, wifi } });
    expect(html).not.toContain(`<strong>${t("信号强")}</strong>`);
    expect(html).not.toContain(`<small>${t("信号强")}</small>`);
  }
}));
