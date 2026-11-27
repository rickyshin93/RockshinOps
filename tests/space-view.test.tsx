import { afterAll, beforeAll, expect, test } from "bun:test";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { I18nextProvider } from "react-i18next";
import { i18n } from "../src/i18n";
import { SpaceView } from "../src/space-view";
import { emptyTrash, invalidateSpace, openFolder, prepareTrash, scanSpace, showTab, spaceStore, stopScan, trashItem } from "../src/space-store";
import type { SpaceListing, SpaceSummary, SpaceTrashOutcome, TrashReview } from "../src/mygo";

const root = { name: "/", path: "/", bytes: 300e9, files: 900, directory: true, other: false, partial: false, trashable: false };
const users = { name: "Users", path: "/Users", bytes: 250e9, files: 800, directory: true, other: false, partial: true, trashable: false };
const iso = { name: "big.iso", path: "/Users/me/Downloads/big.iso", bytes: 8e9, files: 1, directory: false, other: false, partial: false, trashable: true };
const other = { name: "", path: "", bytes: 2e6, files: 12, directory: false, other: true, partial: false, trashable: false };
const summary = {
  id: 10, root: "/", bytes: 300e9, files: 900, skipped: 3, startedAt: "2026-10-30T00:00:00Z", finishedAt: "2026-10-30T00:00:32Z",
  top: [users], largest: [iso],
  suggestions: [{ name: "废纸篓", path: "/Users/me/.Trash", bytes: 5e9, files: 4, found: true, trash: true }],
};
const calls: string[] = [];
const reviewedTrash: TrashReview = { id: 33, path: "/Users/me/.Trash", bytes: 2000, count: 2, items: [{ name: "old", bytes: 2000 }, { name: "another", bytes: 0 }] };
let trashResponse: Promise<SpaceSummary> | undefined;
let purgeResponse: Promise<SpaceTrashOutcome> | undefined;
let prepareResponse: Promise<TrashReview> | undefined;
const discardedReviews: number[] = [];
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
      case "System.TrashSpaceItem": return trashResponse ?? { ...summary, largest: [] };
      case "System.EmptySpaceTrash": return purgeResponse;
      case "System.PrepareSpaceTrash": return prepareResponse;
      case "System.DiscardTrashReview": discardedReviews.push(Number(args[0])); return;
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

test("delayed file actions preserve the receipt without restoring invalidated Space data", async () => {
  await scanSpace();
  const move = Promise.withResolvers<SpaceSummary>();
  trashResponse = move.promise;
  const moving = trashItem(iso);
  invalidateSpace();
  const folderReads = calls.filter(method => method === "System.SpaceListing").length;
  move.resolve({ ...summary, largest: [] });
  await moving;
  trashResponse = undefined;
  expect(spaceStore.get().summary).toBeNull();
  expect(spaceStore.get().notice?.kind).toBe("trashed");
  expect(calls.filter(method => method === "System.SpaceListing").length).toBe(folderReads);

  await scanSpace();
  const purge = Promise.withResolvers<SpaceTrashOutcome>();
  purgeResponse = purge.promise;
  const purging = emptyTrash(reviewedTrash);
  invalidateSpace();
  purge.resolve({
    summary, bytes: 2000, deleted: 1, outcome: { succeeded: [], failed: [
      { path: "folder", changed: true, error: "trash_review_changed", recoveryPath: "/tmp/recovery/folder" },
    ] }, postDeleteFailures: [{ path: "metadata", changed: false, error: "read-only" }],
  });
  await purging;
  purgeResponse = undefined;
  expect(spaceStore.get().summary).toBeNull();
  const html = render();
  expect(html).toContain("Permanently deleted 1 files or folders");
  expect(html).toContain("1 top-level items were not fully deleted");
  expect(html).toContain("The item was kept at /tmp/recovery/folder");
  expect(html).toContain("1 follow-up cleanup steps failed");
});

test("preserve filesystem failure details and localize expired review guidance", async () => {
  await scanSpace();
  await i18n.changeLanguage("zh");
  const purge = Promise.withResolvers<SpaceTrashOutcome>();
  purgeResponse = purge.promise;
  const purging = emptyTrash(reviewedTrash);
  purge.reject(new Error("trash_review_changed"));
  await purging;
  purgeResponse = undefined;
  const html = render();
  expect(html).toContain("废纸篓内容已变化，请重新审阅。");
  expect(html).not.toContain("trash_review_changed");
  await i18n.changeLanguage("en");
});

test("an invalidated or empty preparation releases its backend review", async () => {
  await scanSpace();
  const preparation = Promise.withResolvers<TrashReview>();
  prepareResponse = preparation.promise;
  const inFlight = prepareTrash();
  invalidateSpace();
  preparation.resolve(reviewedTrash);
  expect(await inFlight).toBeUndefined();
  expect(discardedReviews.at(-1)).toBe(reviewedTrash.id);
  expect(spaceStore.get().summary).toBeNull();

  await scanSpace();
  prepareResponse = Promise.resolve({ ...reviewedTrash, id: 34, count: 0, bytes: 0, items: [] });
  expect(await prepareTrash()).toBeUndefined();
  expect(discardedReviews.at(-1)).toBe(34);
  expect(spaceStore.get().notice?.kind).toBe("trashEmpty");
  prepareResponse = undefined;
});
