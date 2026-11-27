import { System, type SpaceEntry, type SpaceListing, type SpaceProgress, type SpaceSummary, type TrashReview, type CleanupFailure } from "./mygo";

export type SpaceTab = "folders" | "files" | "suggestions";
export type SpaceNotice =
  | { kind: "trashed"; name: string; bytes: number }
  | { kind: "purged"; bytes: number; count: number; failed: CleanupFailure[]; postDeleteFailures: CleanupFailure[] }
  | { kind: "trashEmpty" }
  | { kind: "cancelled" }
  | { kind: "error"; message: string };

export interface SpaceState {
  scanning: boolean;
  stopping: boolean;
  progress: SpaceProgress | null;
  progressFailed: boolean;
  summary: SpaceSummary | null;
  listing: SpaceListing | null;
  listingError: string;
  tab: SpaceTab;
  busy: string;
  notice: SpaceNotice | null;
}

// Module state outlives the page component, so a scan keeps running across navigation.
let state: SpaceState = { scanning: false, stopping: false, progress: null, progressFailed: false, summary: null, listing: null, listingError: "", tab: "folders", busy: "", notice: null };
let folderRequest = 0;
let spaceGeneration = 0;
let requestedFolder = "";
let openingFolder = false;
const listeners = new Set<() => void>();

function set(patch: Partial<SpaceState>) {
  state = { ...state, ...patch };
  listeners.forEach(listener => listener());
}

export const spaceStore = {
  subscribe(listener: () => void) { listeners.add(listener); return () => { listeners.delete(listener); }; },
  get: () => state,
};

export function invalidateSpace() {
  spaceGeneration++;
  folderRequest++;
  requestedFolder = "";
  openingFolder = false;
  set({ summary: null, listing: null, listingError: "", notice: null });
}

export async function scanSpace() {
  if (state.scanning || state.busy) return;
  folderRequest++;
  set({ scanning: true, stopping: false, progress: null, progressFailed: false, notice: null });
  const generation = spaceGeneration;
  let active = true;
  let polling = false;
  const timer = setInterval(async () => {
    if (polling) return;
    polling = true;
    try {
      const progress = await System.spaceProgress();
      if (active && progress.scanning) set({ progress, progressFailed: false });
    } catch { if (active) set({ progressFailed: true }); }
    finally { polling = false; }
  }, 500);
  try {
    const summary = await System.scanSpace();
    if (generation !== spaceGeneration) return;
    set({ summary, listing: null, tab: "folders" });
    await loadFolder("");
  } catch (reason) {
    const message = reason instanceof Error ? reason.message : String(reason);
    set({ notice: message === "scan cancelled" ? { kind: "cancelled" } : { kind: "error", message } });
  } finally {
    active = false;
    clearInterval(timer);
    set({ scanning: false, stopping: false, progress: null });
  }
}

export async function stopScan() {
  if (!state.scanning || state.stopping) return;
  set({ stopping: true });
  try { await System.cancelScan(); }
  catch (reason) { set({ stopping: false, notice: { kind: "error", message: reason instanceof Error ? reason.message : String(reason) } }); }
}

async function loadFolder(path: string, open = false) {
  requestedFolder = path;
  openingFolder = open;
  const request = ++folderRequest;
  set({ listingError: "" });
  try {
    const listing = await System.spaceListing(path);
    if (request === folderRequest) {
      openingFolder = false;
      set(open ? { listing, tab: "folders" } : { listing });
    }
  } catch (reason) {
    if (request === folderRequest) {
      openingFolder = false;
      requestedFolder = state.listing?.entry.path ?? "";
      set({ listingError: String(reason) });
    }
  }
}

export function openFolder(path: string) { return loadFolder(path, true); }

export function showTab(tab: SpaceTab) {
  if (openingFolder) {
    folderRequest++;
    openingFolder = false;
    requestedFolder = state.listing?.entry.path ?? "";
  }
  set({ tab });
}

export async function trashItem(entry: SpaceEntry, scanID = state.summary?.id) {
  if (state.busy || state.scanning || scanID == null) return;
  const generation = spaceGeneration;
  set({ busy: entry.path, notice: null });
  try {
    const summary = await System.trashSpaceItem(scanID, entry.path);
    set({ notice: { kind: "trashed", name: entry.name, bytes: entry.bytes } });
    if (generation === spaceGeneration) {
      set({ summary });
      await loadFolder(requestedFolder, openingFolder);
    }
  } catch (reason) { set({ notice: { kind: "error", message: reason instanceof Error ? reason.message : String(reason) } }); }
  finally { set({ busy: "" }); }
}

export async function discardTrashReview(id: number) {
  try { await System.discardTrashReview(id); }
  catch (reason) { set({ notice: { kind: "error", message: reason instanceof Error ? reason.message : String(reason) } }); }
}

export async function prepareTrash(): Promise<TrashReview | undefined> {
  const { summary, busy, scanning } = state;
  if (!summary || busy || scanning) return;
  const generation = spaceGeneration;
  set({ busy: "trash-review", notice: null });
  try {
    const review = await System.prepareSpaceTrash(summary.id);
    if (generation === spaceGeneration && review.count > 0) return review;
    await discardTrashReview(review.id);
    if (generation === spaceGeneration) set({ notice: { kind: "trashEmpty" } });
  } catch (reason) { set({ notice: { kind: "error", message: reason instanceof Error ? reason.message : String(reason) } }); }
  finally { set({ busy: "" }); }
}

export async function emptyTrash(review: TrashReview) {
  if (state.busy || state.scanning) return;
  const generation = spaceGeneration;
  set({ busy: "trash", notice: null });
  try {
    const { summary, outcome, bytes, deleted, postDeleteFailures } = await System.emptySpaceTrash(review.id);
    if (generation === spaceGeneration) {
      if (summary) {
        set({ summary });
        await loadFolder(requestedFolder, openingFolder);
      } else {
        invalidateSpace();
      }
    }
    set({ notice: { kind: "purged", count: deleted, failed: outcome.failed, postDeleteFailures, bytes } });
  } catch (reason) { set({ notice: { kind: "error", message: reason instanceof Error ? reason.message : String(reason) } }); }
  finally { set({ busy: "" }); }
}
