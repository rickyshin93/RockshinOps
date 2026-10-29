import { expect, test } from "bun:test";
import { moveMenuBarItem, reconcileMenuBarState } from "../src/menu-bar-layout";
import type { Item } from "../src/mygo";

const item = (id: number, section: string): Item => ({ id, pid: 42, section, name: String(id), owner: "test", image: "", x: id * 20, width: 20, movable: true });
test("moving an icon updates section and inserts before the target immediately", () => {
  const original = [item(1, "visible"), item(2, "visible"), item(3, "hidden"), item(4, "hidden")];
  const result = moveMenuBarItem(original, 2, "hidden", 4);
  expect(result.filter(i => i.section === "hidden").map(i => i.id)).toEqual([3, 2, 4]);
  expect(result.filter(i => i.section === "visible").map(i => i.id)).toEqual([1]);
  expect(original[1]!.section).toBe("visible");
});
test("dropping on a section appends, and reordering does not duplicate an item", () => {
  const original = [item(1, "visible"), item(2, "hidden"), item(3, "hidden")];
  expect(moveMenuBarItem(original, 1, "hidden").filter(i => i.section === "hidden").map(i => i.id)).toEqual([2, 3, 1]);
  expect(moveMenuBarItem(original, 3, "hidden", 2).filter(i => i.section === "hidden").map(i => i.id)).toEqual([3, 2]);
});
test("revealing an icon previews the native slot immediately beside the divider", () => {
  const original = [item(1, "visible"), item(2, "visible"), item(3, "hidden")];
  expect(moveMenuBarItem(original, 3, "visible").filter(i => i.section === "visible").map(i => i.id)).toEqual([3, 1, 2]);
});

test("unchanged polling retains the rendered state, while visible changes are detected", async () => {
  const { sameMenuBarState } = await import("../src/menu-bar-layout");
  const original = { supported: true, accessibility: true, screenRecording: true, capturing: false, expanded: false, panelVisible: false, iceRunning: false, error: "", items: [item(1, "visible"), item(2, "hidden")] };
  expect(sameMenuBarState(original, structuredClone(original))).toBe(true);
  // Absolute native coordinates can change when the system bar expands; the UI uses order/section.
  expect(sameMenuBarState(original, { ...original, items: original.items.map(i => ({ ...i, x: i.x + 100 })) })).toBe(true);
  expect(sameMenuBarState(original, { ...original, items: [...original.items].reverse() })).toBe(false);
  for (const patch of [{ section: "visible" }, { image: "new image" }, { name: "new name" }, { movable: false }]) {
    expect(sameMenuBarState(original, { ...original, items: [original.items[0]!, { ...original.items[1]!, ...patch }] })).toBe(false);
  }
  expect(sameMenuBarState(original, { ...original, error: "Permission lost" })).toBe(false);
  expect(sameMenuBarState(original, { ...original, accessibility: false })).toBe(false);
  expect(reconcileMenuBarState(original, structuredClone(original))).toBe(original);
  const expanded = reconcileMenuBarState(original, { ...original, expanded: true, items: original.items.map(i => ({ ...i, x: i.x + 100 })) });
  expect(expanded.expanded).toBe(true);
  expect(expanded.items).toBe(original.items);
  const renamed = reconcileMenuBarState(original, { ...original, items: [{ ...original.items[0]!, name: "Renamed" }, original.items[1]!] });
  expect(renamed.items[0]!.name).toBe("Renamed");
  expect(renamed.items).not.toBe(original.items);
});
