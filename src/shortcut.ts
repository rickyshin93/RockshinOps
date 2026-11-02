// Global shortcuts use MyGo accelerator syntax, e.g. "Cmd+Space" or "Ctrl+Alt+K".
const named: Record<string, string> = {
  Space: "Space", Enter: "Enter", Tab: "Tab", Backspace: "Backspace", Delete: "Delete", Escape: "Escape",
  ArrowUp: "Up", ArrowDown: "Down", ArrowLeft: "Left", ArrowRight: "Right", Home: "Home", End: "End", PageUp: "PageUp", PageDown: "PageDown",
  Minus: "-", Equal: "=", Comma: ",", Period: ".", Slash: "/", Semicolon: ";", Quote: "'", BracketLeft: "[", BracketRight: "]", Backslash: "\\", Backquote: "`",
};

type KeyEventLike = Pick<KeyboardEvent, "code" | "metaKey" | "ctrlKey" | "altKey" | "shiftKey">;

/** The accelerator a key press records, or null for a lone modifier, an unknown key or a press without ⌘/⌃/⌥. */
export function acceleratorFromEvent(e: KeyEventLike): string | null {
  const key = /^Key[A-Z]$/.test(e.code) ? e.code.slice(3) : /^Digit\d$/.test(e.code) ? e.code.slice(5) : /^F([1-9]|1\d|2[0-4])$/.test(e.code) ? e.code : named[e.code];
  if (!key || !(e.metaKey || e.ctrlKey || e.altKey)) return null;
  return [e.metaKey && "Cmd", e.ctrlKey && "Ctrl", e.altKey && "Alt", e.shiftKey && "Shift", key].filter(Boolean).join("+");
}

const macSymbols: Record<string, string> = { Cmd: "⌘", Ctrl: "⌃", Alt: "⌥", Shift: "⇧" };

/** How a shortcut reads: "⌃⌥⌘ Space" on macOS (in the system's modifier order), "Ctrl+Alt+Space" elsewhere. */
export function formatShortcut(accelerator: string, isMacOS: boolean): string {
  const parts = accelerator.split("+");
  const key = parts.pop() ?? "";
  if (!isMacOS) return [...parts.map(p => p === "Cmd" ? "Win" : p), key].join("+");
  const order = ["Ctrl", "Alt", "Shift", "Cmd"];
  return `${parts.sort((a, b) => order.indexOf(a) - order.indexOf(b)).map(p => macSymbols[p] ?? p).join("")} ${key}`;
}
