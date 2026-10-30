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
  tab: SpaceTab;
  busy: string;
  notice: SpaceNotice | null;
}

// Module state outlives the page component, so a scan keeps running across navigation.
let state: SpaceState = { scanning: false, stopping: false, progress: null, progressFailed: false, summary: null, listing: null, tab: "folders", busy: "", notice: null };
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
  set({ scanning: true, stopping: false, progress: null, progressFailed: false, notice: null });
  let polling = false;
  const timer = setInterval(async () => {
    if (polling) return;
    polling = true;
    try {
      const progress = await System.spaceProgress();
      if (state.scanning && progress.scanning) set({ progress, progressFailed: false });
    } catch { set({ progressFailed: true }); }
    finally { polling = false; }
  }, 500);
  try {
    const summary = await System.scanSpace();
    set({ summary, listing: await System.spaceListing(""), tab: "folders" });
  } catch (reason) {
    set({ notice: state.stopping ? { kind: "cancelled" } : { kind: "error", message: String(reason) } });
  } finally {
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

export async function openFolder(path: string) {
  try { set({ listing: await System.spaceListing(path), tab: "folders" }); }
  catch (reason) { set({ notice: { kind: "error", message: String(reason) } }); }
}

export function showTab(tab: SpaceTab) { set({ tab }); }

export async function trashItem(entry: SpaceEntry) {
  if (state.busy || state.scanning) return;
  set({ busy: entry.path, notice: null });
  try {
    const summary = await System.trashSpaceItem(entry.path);
    set({ summary, listing: await System.spaceListing(state.listing?.entry.path ?? ""), notice: { kind: "trashed", name: entry.name, bytes: entry.bytes } });
  } catch (reason) { set({ notice: { kind: "error", message: String(reason) } }); }
  finally { set({ busy: "" }); }
}

export async function emptyTrash(bytes: number) {
  if (state.busy || state.scanning) return;
  set({ busy: "trash", notice: null });
  try {
    const summary = await System.emptySpaceTrash();
    set({ summary, listing: await System.spaceListing(state.listing?.entry.path ?? ""), notice: { kind: "emptied", bytes } });
  } catch (reason) { set({ notice: { kind: "error", message: String(reason) } }); }
  finally { set({ busy: "" }); }
}
