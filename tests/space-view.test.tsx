import { afterAll, beforeAll, expect, test } from "bun:test";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { I18nextProvider } from "react-i18next";
import { i18n } from "../src/i18n";
import { SpaceView } from "../src/space-view";
import { openFolder, scanSpace, spaceStore, stopScan } from "../src/space-store";

const root = { name: "/", path: "/", bytes: 300e9, files: 900, directory: true, other: false, partial: false, trashable: false };
const users = { name: "Users", path: "/Users", bytes: 250e9, files: 800, directory: true, other: false, partial: true, trashable: false };
const iso = { name: "big.iso", path: "/Users/me/Downloads/big.iso", bytes: 8e9, files: 1, directory: false, other: false, partial: false, trashable: true };
const other = { name: "", path: "", bytes: 2e6, files: 12, directory: false, other: true, partial: false, trashable: false };
const summary = {
  root: "/", bytes: 300e9, files: 900, skipped: 3, startedAt: "2026-10-30T00:00:00Z", finishedAt: "2026-10-30T00:00:32Z",
  top: [users], largest: [iso],
  suggestions: [{ name: "废纸篓", path: "/Users/me/.Trash", bytes: 5e9, files: 4, found: true, trash: true }],
};
const calls: string[] = [];
let pending: { reject: (error: Error) => void } | null = null;
const volume = { name: "Macintosh HD", mount: "/", format: "apfs", total: 500e9, available: 150e9 };
const previous = { mygo: (globalThis as any).mygo, document: globalThis.document };

beforeAll(async () => {
  globalThis.document = { createElementNS: () => ({ setAttribute() {}, appendChild() {}, outerHTML: "<svg></svg>" }) } as unknown as Document;
  (globalThis as any).mygo = { call: async (method: string, ...args: unknown[]) => {
    calls.push(method);
    switch (method) {
      case "System.ScanSpace": return pending ? new Promise((_, reject) => { pending = { reject }; }) : summary;
      case "System.SpaceListing": return args[0] === "/Users" ? { entry: users, ancestors: [root], children: [iso, other] } : { entry: root, ancestors: [], children: [users, other] };
      case "System.CancelScan": pending?.reject(new Error("scan cancelled")); return;
      case "System.SpaceProgress": return { scanning: true, files: 10, bytes: 1e9, skipped: 0, path: "/Users" };
      default: throw new Error(method);
    }
  } };
  await i18n.changeLanguage("en");
});
afterAll(() => { (globalThis as any).mygo = previous.mygo; globalThis.document = previous.document; });

const render = () => renderToStaticMarkup(createElement(I18nextProvider, { i18n }, createElement(SpaceView, { volume, capacityError: "", isMacOS: true, openDiskAccess() {} })));

test("one scan feeds the breakdown, drill-down, large files and stop without rescanning", async () => {
  expect(render()).toContain("Start scan");
  await scanSpace();
  let html = render();
  expect(html).toContain("Space breakdown");
  expect(html).toContain('title="/Users"');
  expect(html).toContain("partly unreadable");
  expect(html).toContain("12 smaller items");
  expect(html).toContain("3 locations could not be read");
  expect(html).toContain("System data &amp; uncounted");

  await openFolder("/Users");
  html = render();
  expect(html).toContain("Macintosh HD");
  expect(html).toContain('aria-label="Move “big.iso” to Trash"');
  expect(html.match(/aria-label="Move /g)?.length).toBe(1);
  expect(calls.filter(call => call === "System.ScanSpace").length).toBe(1);

  pending = { reject() {} };
  const running = scanSpace();
  await stopScan();
  await running;
  expect(spaceStore.get().summary).toBe(summary as never);
  expect(render()).toContain("Scan stopped; the previous result is kept.");
});
