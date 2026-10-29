import { expect, test } from "bun:test";
import { connectionRows, nextConnectionSort, splitEndpoint } from "../src/connections";
import { actionControls, connectionSubtitle, renderBody, type ViewState } from "../src/views";
import type { Connection, Process } from "../src/mygo";
import { t } from "../src/i18n";

test.each([
  ["127.0.0.1:443", "127.0.0.1", "443"],
  ["[fe80::1%en0]:5173", "fe80::1%en0", "5173"],
  ["*:80", "*", "80"],
  ["example.local", "example.local", ""],
  ["", "", ""],
])("separates native endpoint %s without assuming its service", (endpoint, host, port) => {
  expect(splitEndpoint(endpoint)).toEqual({ host, port });
});

const connection: Connection = { pid: 42, process: "Chrome He", userId: "501", protocol: "TCP", family: "IPv6", local: "[::1]:54321", remote: "[2001:db8::1]:443", state: "ESTABLISHED" };
const process = { pid: 42, name: "Chrome Helper", app: "/Applications/Chrome.app", path: "/Applications/Chrome.app/Contents/Helper" } as Process;

test("header sorting toggles direction and sorts numeric ports with missing values last", () => {
  const items = [443, 80, 9].map((port, index) => ({ ...connection, pid: index, remote: `127.0.0.1:${port}` }));
  items.push({ ...connection, pid: 3, remote: "", local: "host" });
  const ascending = nextConnectionSort(null, "port");
  expect(connectionRows(items, [], "", ascending).map(row => row.pid)).toEqual([2, 1, 0, 3]);
  const descending = nextConnectionSort(ascending, "port");
  expect(connectionRows(items, [], "", descending).map(row => row.pid)).toEqual([0, 1, 2, 3]);
  expect(nextConnectionSort(descending, "process")).toEqual({ key: "process", direction: "ascending" });
  expect(connectionRows(items, [], "443", descending)).toHaveLength(1);
  expect(items.map(row => row.pid)).toEqual([0, 1, 2, 3]);
});

test.each(["process", "host", "protocol", "state"] as const)("sorts %s in both directions", key => {
  const items = [
    { ...connection, pid: 1, process: "Zulu", remote: "10.0.0.10:80", protocol: "UDP", state: "LISTEN" },
    { ...connection, pid: 2, process: "Alpha", remote: "10.0.0.2:80", protocol: "TCP", state: "ESTABLISHED" },
  ];
  expect(connectionRows(items, [], "", { key, direction: "ascending" }).map(row => row.pid)).toEqual([2, 1]);
  expect(connectionRows(items, [], "", { key, direction: "descending" }).map(row => row.pid)).toEqual([1, 2]);
});

test("uses exact PID metadata for name and icon, and filters complete names and endpoints", () => {
  expect(connectionRows([connection], [process], "chrome helper")).toMatchObject([{ process: "Chrome Helper", path: "/Applications/Chrome.app" }]);
  expect(connectionRows([connection], [], " 2001:db8::1 ")).toHaveLength(1);
  expect(connectionRows([connection], [], "42")).toHaveLength(1);
  expect(connectionRows([connection], [{ ...process, pid: 43 }], "helper")).toHaveLength(0);
  expect(connectionRows([connection], [], "")[0]).toMatchObject({ process: "Chrome He", path: "" });
});

test("connection view keeps unavailable telemetry explicit and exposes pause without metric ranges", () => {
  const previousDocument = globalThis.document;
  globalThis.document = { createElementNS: () => ({ setAttribute() {}, appendChild() {}, outerHTML: "<svg></svg>" }) } as unknown as Document;
  try {
    const state = { page: "connections", connections: { items: [connection], updatedAt: "2026-10-29T01:00:00Z", warning: "" }, processes: { items: [process] }, info: { machine: { platform: "darwin" } }, snapshot: {}, connectionError: "", query: "", paused: false } as ViewState;
    const html = renderBody(state);
    expect(html.match(/<th scope=/g)).toHaveLength(10);
    expect(html.match(/data-connection-sort=/g)).toHaveLength(5);
    expect(renderBody({ ...state, connectionSort: { key: "port", direction: "descending" } })).toContain('aria-sort="descending"');
    expect(html.match(/class="connection-unavailable"/g)).toHaveLength(5);
    expect(html).not.toContain("0 B/s");
    expect(html).toContain("2001:db8::1");
    expect(html).toContain("appicon://localhost/?path=%2FApplications%2FChrome.app");
    expect(html).toContain("PID 42");
    expect(html).toContain("[::1]:54321");
    expect(actionControls(state)).toContain('data-action="pause"');
    expect(actionControls(state)).not.toContain("data-range=");
    expect(actionControls({ ...state, paused: true })).toContain('aria-pressed="true"');
    expect(connectionSubtitle({ ...state, query: "no match" })).toContain("0 / 1");
    expect(renderBody({ ...state, query: "no match" })).not.toContain('<table');
    expect(renderBody({ ...state, connectionError: "Read failed" })).toContain("Read failed");
  } finally { globalThis.document = previousDocument; }
});

test("process details show measured values and handle a process leaving the snapshot", () => {
  const previousDocument = globalThis.document;
  globalThis.document = { createElementNS: () => ({ setAttribute() {}, appendChild() {}, outerHTML: "<svg></svg>" }) } as unknown as Document;
  try {
    const item = { ...process, identity: "123456", cpu: 12, cpuReady: true, memory: 1024, threads: 3, userId: 501, ioReady: false, own: true } as Process;
    const state = { page: "processes", snapshot: {}, processes: { items: [item], total: 1, skipped: 0, updatedAt: "2026-10-29T01:00:00Z" }, selectedProcess: 42, info: { machine: { platform: "darwin" } }, query: "", sort: "cpu", paused: false } as ViewState;
    const html = renderBody(state);
    expect(html).toContain('data-select-process="42" aria-pressed="true"');
    expect(html).toContain("PID 42");
    expect(html).toContain("501");
    expect(html).toContain("/Applications/Chrome.app/Contents/Helper");
    expect(html).toContain(t("当前采集源未提供单进程 GPU、功率或 CPU 时间；用户列显示 UID。"));
    expect(html).not.toContain("data-action=\"kill");
    expect(html).toContain('data-action="quit-process"  title=');
    expect(html).toContain('data-action="force-quit-process"  class=');
    const pending = renderBody({ ...state, processAction: { pid: 42, busy: true, error: false, message: "Sending" } });
    expect(pending).toContain('data-action="quit-process" disabled');
    expect(pending).toContain('data-action="force-quit-process" disabled');
    expect(pending).toContain('role="status">Sending');
    expect(renderBody({ ...state, processAction: { pid: 42, busy: false, error: true, message: "Permission denied" } })).toContain('role="alert">Permission denied');
    expect(actionControls(state)).toContain('data-action="pause"');
    expect(renderBody({ ...state, processes: { ...state.processes!, items: [] } })).toContain(t("进程可能已退出或不再可见。"));
    const scoped = { ...state, selectedProcess: null, processes: { ...state.processes!, items: [item, { ...item, pid: 9, name: "daemon", app: "", own: false, memory: 2048 }, { ...item, pid: 7, name: "shell", app: "" }] } };
    const visiblePids = (value: ViewState) => [...renderBody(value).matchAll(/<tr[^>]*data-select-process="(\d+)"/g)].map(match => Number(match[1]));
    expect(visiblePids({ ...scoped, processScope: "apps" })).toEqual([42]);
    expect(visiblePids({ ...scoped, processScope: "mine" })).toEqual([7, 42]);
    expect(visiblePids({ ...scoped, processScope: "system" })).toEqual([9]);
    expect(visiblePids({ ...scoped, processOrder: { key: "memory", descending: true } })).toEqual([9, 7, 42]);
    expect(visiblePids({ ...scoped, processOrder: { key: "pid", descending: false } })).toEqual([7, 9, 42]);
    expect(visiblePids({ ...scoped, query: "shell" })).toEqual([7]);
  } finally { globalThis.document = previousDocument; }
});
