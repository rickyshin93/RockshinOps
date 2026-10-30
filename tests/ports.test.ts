import { expect, test } from "bun:test";
import { addressScope, filterPorts, listeningPorts } from "../src/ports";
import { portsView } from "../src/ports-view";
import type { Connection, Process } from "../src/mygo";
import type { ViewState } from "../src/views";
import { t } from "../src/i18n";

const listener: Connection = { pid: 42, process: "node", userId: "501", protocol: "TCP", family: "IPv4", local: "127.0.0.1:3000", remote: "", state: "LISTEN" };

test("only TCP listeners enter the workbench; dual-stack bindings combine but different owners remain distinct", () => {
  const rows = listeningPorts([
    listener, listener, { ...listener, local: "[::1]:3000", family: "IPv6" },
    { ...listener, pid: 43 }, { ...listener, local: "*:80" },
    { ...listener, local: "*:99999" }, { ...listener, local: "*:0" },
    { ...listener, protocol: "UDP", local: "*:3000" },
    { ...listener, state: "ESTABLISHED", remote: "10.0.0.1:443" },
  ], []);
  expect(rows.map(row => [row.port, row.pid])).toEqual([[80, 42], [3000, 42], [3000, 43]]);
  expect(rows[1]!.addresses).toEqual(["127.0.0.1:3000", "[::1]:3000"]);
  expect(rows[1]!.identity).toBe("");
});

test.each([
  ["127.0.0.1", "loopback"], ["127.1.2.3", "loopback"], ["::1", "loopback"],
  ["::ffff:127.0.0.1", "loopback"], ["*", "wildcard"], ["0.0.0.0", "wildcard"],
  ["::", "wildcard"], ["192.168.1.12", "interface"], ["fe80::1%en0", "interface"],
] as const)("classifies binding %s without claiming network reachability", (host, expected) => {
  expect(addressScope(host)).toBe(expected);
});

test("numeric searches match exact local port or PID; names and addresses remain searchable", () => {
  const rows = listeningPorts([listener, { ...listener, pid: 3000, local: "*:9000" }, { ...listener, pid: 90, local: "*:30001" }], [{ pid: 42, name: "Web dev server", app: "/Applications/Editor.app" } as Process]);
  expect(filterPorts(rows, "3000", "all").map(row => row.port)).toEqual([3000, 9000]);
  expect(filterPorts(rows, ":3000", "all").map(row => row.port)).toEqual([3000]);
  expect(filterPorts(rows, "EDITOR", "loopback")).toHaveLength(1);
  expect(filterPorts(rows, "127.0.0.1", "wildcard")).toHaveLength(0);
  expect(filterPorts(rows, " 42 ", "all")).toHaveLength(1);
  expect(filterPorts(rows, "", "wildcard")).toHaveLength(2);
});

test("port view escapes native values and keeps favorite, empty, stale and failure states explicit", () => {
  const before = globalThis.document;
  globalThis.document = { createElementNS: () => ({ setAttribute() {}, appendChild() {}, outerHTML: "<svg></svg>" }) } as unknown as Document;
  try {
    const state = { connections: { items: [{ ...listener, process: '<img src=x onerror="bad">' }], warning: "" }, query: "", portScope: "all", favoritePorts: [3000, 5432], portsLoaded: true } as ViewState;
    const html = portsView(state);
    expect(html).toContain('&lt;img');
    expect(html).not.toContain('<img src=x');
    expect(html).toContain('data-copy-port="127.0.0.1:3000"');
    expect(html).toContain('data-port-process="42"');
    expect(html).toContain(t("未发现监听"));
    expect(html).toContain(t("本机回环"));
    expect(html).toContain('class="icon-button port-close" disabled');
    const owned = { ...state, processes: { items: [{ pid: 42, identity: "start-42", name: "node" } as Process] } } as ViewState;
    expect(portsView(owned)).toContain('data-close-port="42"');
    expect(portsView(owned)).toContain('app-icon process-system-icon');
    const withApp = { ...owned, info: { machine: { platform: "darwin" } }, processes: { items: [{ pid: 42, identity: "start-42", name: "node", app: "/Applications/My App.app" } as Process] } } as ViewState;
    expect(portsView(withApp)).toContain('appicon://localhost/?path=%2FApplications%2FMy%20App.app');
    expect(portsView({ ...owned, portAction: { pid: 42, busy: true } })).toContain('data-close-port="42" aria-label="Close port 3000" title="Close port 3000" disabled aria-busy="true"');
    expect(portsView({ ...state, connections: { items: [{ ...listener, local: "*:3000" }], warning: "" } })).toContain(t("所有接口"));
    expect(portsView({ ...state, query: "65535" })).toContain(t("没有匹配的监听端口"));
    expect(portsView({ ...state, query: "65535" })).toContain('data-action="clear-port-filters"');
    const stale = portsView({ ...state, connectionError: "lsof failed" });
    expect(stale).toContain(t("更新中断 · 上次数据"));
    expect(stale).toContain(t("状态未知"));
    expect(stale).toContain('data-copy-port="127.0.0.1:3000"');
    expect(stale).toContain(t("连接数据暂不可用"));
    expect(portsView({ ...state, portsError: "save failed", portsLoaded: false })).toContain('disabled');
    expect(portsView({ ...state, connections: null, connectionError: "read failed" })).toContain(t("无法读取连接"));
  } finally { globalThis.document = before; }
});

test("same wildcard endpoint on IPv4 and IPv6 preserves both bindings", () => {
  const rows = listeningPorts([{ ...listener, local: "*:5000" }, { ...listener, local: "*:5000", family: "IPv6" }], []);
  expect(rows).toHaveLength(1);
  expect(rows[0]!.addresses).toEqual(["0.0.0.0:5000", "[::]:5000"]);
});
