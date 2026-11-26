import { afterAll, beforeAll, expect, test } from "bun:test";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { I18nextProvider } from "react-i18next";
import { i18n } from "../src/i18n";
import { SpaceView } from "../src/space-view";
import { openFolder, scanSpace, showTab, spaceStore, stopScan, trashItem } from "../src/space-store";
import type { SpaceListing } from "../src/mygo";

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
let cancellationError = "scan cancelled";
let listingOverride: ((path: string) => Promise<SpaceListing> | undefined) | undefined;
const volume = { name: "Macintosh HD", mount: "/", format: "apfs", total: 500e9, available: 150e9 };
const previous = { mygo: (globalThis as any).mygo, document: globalThis.document };

beforeAll(async () => {
  globalThis.document = { createElementNS: () => ({ setAttribute() {}, appendChild() {}, outerHTML: "<svg></svg>" }) } as unknown as Document;
  (globalThis as any).mygo = { callWith: async (method: string, args: unknown[]) => {
    calls.push(method);
    switch (method) {
      case "System.ScanSpace": return pending ? new Promise((_, reject) => { pending = { reject }; }) : summary;
      case "System.SpaceListing": return listingOverride?.(String(args[0])) ?? (args[0] === "/Users" ? { entry: users, ancestors: [root], children: [iso, other] } : { entry: root, ancestors: [], children: [users, other] });
      case "System.CancelScan": pending?.reject(new Error(cancellationError)); return;
      case "System.TrashSpaceItem": return { ...summary, largest: [] };
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
  pending = null;
  expect(spaceStore.get().summary).toBe(summary as never);
  expect(render()).toContain("Scan stopped; the previous result is kept.");
});

test("folder responses respect the latest folder or tab selection", async () => {
  await scanSpace();
  const older = Promise.withResolvers<SpaceListing>();
  const newer = Promise.withResolvers<SpaceListing>();
  listingOverride = path => path === "/Users" ? older.promise : newer.promise;
  const first = openFolder("/Users");
  const second = openFolder("/");
  newer.resolve({ entry: root, ancestors: [], children: [users] });
  await second;
  older.resolve({ entry: users, ancestors: [root], children: [] });
  await first;
  expect(spaceStore.get().listing?.entry.path).toBe("/");
  const late = Promise.withResolvers<SpaceListing>();
  listingOverride = () => late.promise;
  const opening = openFolder("/Users");
  showTab("files");
  late.resolve({ entry: users, ancestors: [root], children: [] });
  await opening;
  expect(spaceStore.get().tab).toBe("files");
  listingOverride = undefined;
});

test("a successful trash move survives a failed folder refresh", async () => {
  await scanSpace();
  listingOverride = async () => { throw new Error("listing failed"); };
  await trashItem(iso);
  listingOverride = undefined;
  expect(spaceStore.get().summary?.largest).toEqual([]);
  expect(spaceStore.get().notice?.kind).toBe("trashed");
  expect(render()).toContain("listing failed");
});

test("a real scan error is visible even when stop was requested", async () => {
  pending = { reject() {} };
  cancellationError = "EACCES";
  const running = scanSpace();
  await stopScan();
  await running;
  pending = null;
  cancellationError = "scan cancelled";
  expect(spaceStore.get().notice).toEqual({ kind: "error", message: "EACCES" });
});
