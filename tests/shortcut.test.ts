import { expect, test } from "bun:test";
import { acceleratorFromEvent, formatShortcut } from "../src/shortcut";

const press = (code: string, mods: Partial<Record<"metaKey" | "ctrlKey" | "altKey" | "shiftKey", boolean>> = {}) =>
  acceleratorFromEvent({ code, metaKey: false, ctrlKey: false, altKey: false, shiftKey: false, ...mods });

test("records accelerators MyGo can register", () => {
  expect(press("Space", { metaKey: true })).toBe("Cmd+Space");
  expect(press("KeyK", { ctrlKey: true, altKey: true, shiftKey: true })).toBe("Ctrl+Alt+Shift+K");
  expect(press("Digit1", { altKey: true })).toBe("Alt+1");
  expect(press("F12", { metaKey: true })).toBe("Cmd+F12");
  expect(press("Slash", { metaKey: true })).toBe("Cmd+/");
});

test("ignores lone modifiers, unknown keys and presses without a command modifier", () => {
  expect(press("MetaLeft", { metaKey: true })).toBeNull();
  expect(press("KeyA")).toBeNull();
  expect(press("KeyA", { shiftKey: true })).toBeNull();
  expect(press("IntlRo", { metaKey: true })).toBeNull();
});

test("formats shortcuts per platform", () => {
  expect(formatShortcut("Cmd+Space", true)).toBe("⌘ Space");
  expect(formatShortcut("Cmd+Ctrl+Alt+Shift+K", true)).toBe("⌃⌥⇧⌘ K");
  expect(formatShortcut("Alt+Space", false)).toBe("Alt+Space");
  expect(formatShortcut("Cmd+K", false)).toBe("Win+K");
});
