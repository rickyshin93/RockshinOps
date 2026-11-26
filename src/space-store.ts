import { System, type SpaceEntry, type SpaceListing, type SpaceProgress, type SpaceSummary } from "./mygo";

export type SpaceTab = "folders" | "files" | "suggestions";
export type SpaceNotice =
  | { kind: "trashed"; name: string; bytes: number }
  | { kind: "emptied"; bytes: number }
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

export async function scanSpace() {
  if (state.scanning || state.busy) return;
  folderRequest++;
  set({ scanning: true, stopping: false, progress: null, progressFailed: false, notice: null });
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
  catch (reason) { set({ stopping: false, notice: { kind: "error", message: String(reason) } }); }
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

export async function trashItem(entry: SpaceEntry) {
  if (state.busy || state.scanning) return;
  set({ busy: entry.path, notice: null });
  try {
    const summary = await System.trashSpaceItem(entry.path);
    set({ summary, notice: { kind: "trashed", name: entry.name, bytes: entry.bytes } });
    await loadFolder(requestedFolder, openingFolder);
  } catch (reason) { set({ notice: { kind: "error", message: String(reason) } }); }
  finally { set({ busy: "" }); }
}

export async function emptyTrash(bytes: number) {
  if (state.busy || state.scanning) return;
  set({ busy: "trash", notice: null });
  try {
    const summary = await System.emptySpaceTrash();
    set({ summary, notice: { kind: "emptied", bytes } });
    await loadFolder(requestedFolder, openingFolder);
  } catch (reason) { set({ notice: { kind: "error", message: String(reason) } }); }
  finally { set({ busy: "" }); }
}
