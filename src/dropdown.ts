import { t } from "./i18n";
import { icon } from "./icons";

export function sortControl(value: string, processes: boolean): string {
  const choices = [["cpu", t("CPU 优先")], ["memory", t("内存优先")], ...(processes ? [["readRate", t("读取优先")], ["writeRate", t("写入优先")]] : []), ["name", t("名称排序")]];
  return `<div class="sort-control"><button id="sort-trigger" type="button" role="combobox" aria-label="${t("排序方式")}" aria-haspopup="listbox" aria-expanded="false" aria-controls="sort-options">${choices.find(([id]) => id === value)![1]}${icon("down")}</button><div id="sort-options" class="sort-menu" role="listbox" aria-label="${t("排序方式")}" hidden>${choices.map(([id, label]) => `<button type="button" role="option" tabindex="-1" aria-selected="${id === value}" data-sort="${id}"><span>${label}</span>${id === value ? icon("check") : ""}</button>`).join("")}</div></div>`;
}

export function installSortControl(onSelect: (value: string) => void) {
  const trigger = () => document.getElementById("sort-trigger");
  const menu = () => document.getElementById("sort-options");
  const options = () => Array.from(menu()?.querySelectorAll<HTMLButtonElement>("[role=option]") || []);
  function close(focus = false) {
    const m = menu(); if (!m) return;
    m.hidden = true; trigger()?.setAttribute("aria-expanded", "false");
    if (focus) trigger()?.focus();
  }
  function open() {
    const m = menu(); if (!m) return;
    m.hidden = false; trigger()?.setAttribute("aria-expanded", "true");
    (options().find(o => o.getAttribute("aria-selected") === "true") || options()[0])?.focus();
  }
  document.addEventListener("click", event => {
    const target = event.target as Element;
    if (target.closest("#sort-trigger")) { if (menu()?.hidden) open(); else close(true); return; }
    const option = target.closest<HTMLElement>("[data-sort]");
    if (option) { onSelect(option.dataset.sort!); trigger()?.focus(); return; }
    if (!target.closest(".sort-control")) close();
  });
  document.addEventListener("keydown", event => {
    const target = event.target as Element;
    if (!target.closest(".sort-control")) return;
    if (event.key === "Tab") { close(); return; }
    if (event.key === "Escape") { event.preventDefault(); close(true); return; }
    if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return;
    event.preventDefault();
    if (menu()?.hidden) { open(); return; }
    const items = options(), index = items.indexOf(document.activeElement as HTMLButtonElement);
    const next = event.key === "Home" ? 0 : event.key === "End" ? items.length - 1 : (index + (event.key === "ArrowDown" ? 1 : -1) + items.length) % items.length;
    items[next]?.focus();
  });
  document.addEventListener("focusin", event => { if (!(event.target as Element).closest(".sort-control")) close(); });
}
