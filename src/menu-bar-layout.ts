import type { Item, State } from "./mygo";

// Expansion moves native windows, but does not change their preview content.
export function reconcileMenuBarState(previous: State, next: State): State {
  if (sameMenuBarState(previous, next)) return previous;
  return sameMenuBarState({ ...next, items: previous.items }, next)
    ? { ...next, items: previous.items } : next;
}

// Native polling returns fresh objects even when nothing displayed has changed.
export function sameMenuBarState(a: State, b: State): boolean {
  return a.supported === b.supported && a.accessibility === b.accessibility
    && a.screenRecording === b.screenRecording && a.capturing === b.capturing
    && a.expanded === b.expanded && a.panelVisible === b.panelVisible
    && a.iceRunning === b.iceRunning && a.error === b.error
    && a.items.length === b.items.length && a.items.every((item, index) => {
      const next = b.items[index]!;
      return item.id === next.id && item.pid === next.pid && item.section === next.section
        && item.name === next.name && item.owner === next.owner && item.image === next.image
        && item.width === next.width && item.movable === next.movable;
    });
}

// Preview the exact move requested from macOS; the native result confirms it.
export function moveMenuBarItem(items: Item[], id: number, section: string, before = 0): Item[] {
  const original = items.find(item => item.id === id)!;
  const moving = { ...original, section };
  const remaining = items.filter(item => item.id !== id);
  const target = before ? remaining.findIndex(item => item.id === before)
    : section === "visible" && original.section !== section ? remaining.findIndex(item => item.section === section) : -1;
  const last = remaining.reduce((index, item, current) => item.section === section ? current : index, -1);
  remaining.splice(target >= 0 ? target : last + 1, 0, moving);
  return remaining;
}
