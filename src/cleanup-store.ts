import { System, type CleanupOutcome, type CleanupProgress, type CleanupScan } from "./mygo";
import { invalidateSpace } from "./space-store";

type CleanupState = {
  mode: "quick" | "space";
  scan: CleanupScan | null;
  selected: ReadonlySet<string>;
  scanning: boolean;
  stopping: boolean;
  progress: CleanupProgress | null;
  moving: boolean;
  cancelled: boolean;
  error: string;
  outcome: CleanupOutcome | null;
};

let state: CleanupState = {
  mode: "quick", scan: null, selected: new Set(), scanning: false, stopping: false,
  progress: null, moving: false, cancelled: false, error: "", outcome: null,
};
const listeners = new Set<() => void>();

function set(patch: Partial<CleanupState>) {
  state = { ...state, ...patch };
  listeners.forEach(listener => listener());
}

export const cleanupStore = {
  subscribe(listener: () => void) { listeners.add(listener); return () => { listeners.delete(listener); }; },
  get: () => state,
};

export function showCleanupMode(mode: CleanupState["mode"]) { set({ mode }); }

function recommended(scan: CleanupScan) {
  return new Set(scan.categories.flatMap(category => category.items.filter(item => item.recommended).map(item => item.path)));
}

export function restoreRecommended() {
  if (state.scan) set({ selected: recommended(state.scan) });
}

export function selectCleanup(paths: string[], checked: boolean) {
  if (state.moving || state.scanning) return;
  const selected = new Set(state.selected);
  for (const path of paths) {
    if (checked) selected.add(path);
    else selected.delete(path);
  }
  set({ selected });
}

export async function scanCleanup() {
  if (state.scanning || state.moving) return;
  set({ scanning: true, stopping: false, progress: null, cancelled: false, error: "", outcome: null });
  let active = true;
  let polling = false;
  const timer = setInterval(async () => {
    if (polling) return;
    polling = true;
    try {
      const progress = await System.cleanupProgress();
      if (active && progress.total > 0 && progress.category !== state.progress?.category) set({ progress });
    } catch (error) {
      if (active) set({ error: String(error) });
    } finally { polling = false; }
  }, 250);
  try {
    const scan = await System.scanCleanup();
    set({ scan, selected: recommended(scan), error: "" });
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    set(message === "scan cancelled" ? { cancelled: true } : { error: message });
  } finally {
    active = false;
    clearInterval(timer);
    set({ scanning: false, stopping: false, progress: null });
  }
}

export async function stopCleanup() {
  if (!state.scanning || state.stopping || !state.progress) return;
  set({ stopping: true });
  try { await System.cancelCleanupScan(); }
  catch (error) { set({ stopping: false, error: String(error) }); }
}

export async function trashSelectedCleanup() {
  const { scan, selected } = state;
  if (!scan || !selected.size || state.moving || state.scanning) return;
  set({ moving: true, error: "", outcome: null });
  try {
    const outcome = await System.trashCleanup(scan.id, [...selected]);
    const removed = new Set([...outcome.moved.map(item => item.path), ...outcome.failed.filter(item => item.changed).map(item => item.path)]);
    if (outcome.moved.length) invalidateSpace();
    set({
      scan: { ...scan, categories: scan.categories.map(category => ({ ...category, items: category.items.filter(item => !removed.has(item.path)) })) },
      selected: new Set([...selected].filter(path => !removed.has(path))), outcome,
    });
  } catch (error) { set({ error: String(error) }); }
  finally { set({ moving: false }); }
}

export async function revealCleanup(path: string) {
  if (!state.scan) return;
  try { await System.revealCleanupItem(state.scan.id, path); }
  catch (error) { set({ error: String(error) }); }
}
