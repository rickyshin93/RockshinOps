import { escapeHTML as e } from "./format";
import { t } from "./i18n";
import { icon } from "./icons";
import type { LauncherSearch } from "./mygo";
import { clipboardImageURL, clipboardWarningKey } from "./clipboard-ui";

export interface ClipboardViewState {
  clipboardHistory?: LauncherSearch | null;
  clipboardMode?: "history" | "snippets";
  clipboardBusy?: boolean;
  clipboardReadError?: boolean;
  clipboardError?: string;
  clipboardCopiedId?: string;
}

export function clipboardControls(state: ClipboardViewState): string {
  if (!state.clipboardHistory || state.clipboardMode === "snippets") return "";
  const disabled = state.clipboardBusy ? 'aria-disabled="true"' : "";
  const pinned = state.clipboardHistory.clipboardPinnedCount ?? 0;
  const label = e(t(pinned ? "清空未置顶历史" : "清空历史"));
  const clearable = (state.clipboardHistory.clipboardCount ?? 0) > pinned;
  return `<button type="button" class="icon-button clipboard-action" data-action="clear-clipboard" aria-label="${label}" title="${label}" ${clearable ? disabled : "disabled"}>${icon("trash")}</button>`;
}

export function clipboardFilters(state: ClipboardViewState, query: string): string {
  const snippets = state.clipboardMode === "snippets";
  const search = e(t(snippets ? "搜索片段" : "搜索剪贴板历史"));
  return `<div class="segmented" role="group" aria-label="${e(t("剪贴板视图"))}">
    <button type="button" data-action="clipboard-show-history" aria-pressed="${!snippets}" class="${snippets ? "" : "selected"}">${e(t("剪贴板历史"))}</button>
    <button type="button" data-action="clipboard-show-snippets" aria-pressed="${snippets}" class="${snippets ? "selected" : ""}">${e(t("片段"))}</button>
  </div><label class="search-field">${icon("search")}<input id="search" type="search" placeholder="${e(t("搜索"))}" aria-label="${search}" value="${e(query)}" autocomplete="off" spellcheck="false"/></label>`;
}

export function clipboardView(state: ClipboardViewState, query: string): string {
  const history = state.clipboardHistory;
  const snippets = state.clipboardMode === "snippets";
  const warning = clipboardWarningKey(history?.clipboardWarning);
  const copiedLabel = e(t("已复制到剪贴板"));
  const announcement = state.clipboardCopiedId && !state.clipboardBusy ? copiedLabel : "";
  const notices = [
    state.clipboardReadError ? `<div class="notice error" role="alert">${e(t("无法读取剪贴板历史。"))} <button type="button" class="secondary-button" data-action="load-clipboard">${e(t("重试"))}</button></div>` : "",
    state.clipboardError ? `<p class="notice error" role="alert">${e(t(state.clipboardError))}</p>` : "",
    !snippets && history?.clipboardStatus === "permission" ? `<p class="notice warning" role="status">${e(t("请在 macOS 系统设置中始终允许 RockshinOps 访问剪贴板。"))}</p>` : "",
    !snippets && warning ? `<p class="notice warning" role="status">${e(t(warning))}</p>` : "",
  ].join("");
  const feedback = `<p class="sr-only" role="status">${announcement}</p><div>${notices}</div>`;
  if (!history) return `${feedback}${notices ? "" : `<div class="empty-state" role="status"><p>${e(t("正在加载页面…"))}</p></div>`}`;
  const results = snippets ? history.clipboardSnippets ?? [] : history.items;
  if (!results.length) {
    if (!snippets && history.clipboardStatus === "permission") return feedback;
    const empty = snippets ? "在剪贴板历史中，将常用文本另存为片段。" : "复制文本或图片，即可在这里找回。";
    return `${feedback}<div class="empty-state" role="status"><p>${e(t(query.trim() ? "没有找到匹配的结果" : empty))}</p></div>`;
  }
  const copy = e(t("复制"));
  const pin = e(t("置顶"));
  const save = e(t("另存为片段"));
  const disabled = state.clipboardBusy ? 'aria-disabled="true"' : "";
  const items = results.map(item => {
    const copied = state.clipboardCopiedId === item.id;
    const id = e(item.id);
    const pinned = !!item.clipboardPinned;
    return `<li data-row-key="${id}">
    ${item.clipboardImage
      ? `<button type="button" class="clipboard-history-thumbnail" data-clipboard-preview="${e(item.id)}" aria-label="${e(t("预览图片"))}" aria-haspopup="dialog"><img src="${e(clipboardImageURL(item.id))}" loading="lazy" alt=""></button>`
      : `<span class="clipboard-history-icon">${icon(snippets ? "snippet" : "clipboard")}</span>`}
    <div class="clipboard-history-preview"><p title="${e(item.title)}">${e(item.title)}</p><small>${e(item.subtitle)}</small></div>
    <div class="clipboard-row-actions">
      ${snippets || item.clipboardImage ? "" : `<button type="button" class="icon-button clipboard-action" data-clipboard-snippet="${id}" data-focus-key="snippet:${id}" aria-label="${save}" title="${save}" ${disabled}>${icon("save-snippet")}</button>`}
      ${snippets ? "" : `<button type="button" class="icon-button clipboard-action${pinned ? " is-pinned" : ""}" data-clipboard-pin="${id}" data-focus-key="pin:${id}" aria-pressed="${pinned}" aria-label="${pin}" title="${pinned ? e(t("取消置顶")) : pin}" ${disabled}>${icon("pin")}</button>`}
      <button type="button" class="icon-button clipboard-action${copied ? " is-copied" : ""}" data-clipboard-id="${id}" aria-label="${copy}" title="${copy}" ${disabled}>${icon(copied ? "check" : "copy")}</button>
    </div>
  </li>`;
  }).join("");
  return `${feedback}<ul class="clipboard-history-list${snippets ? " clipboard-snippets-list" : ""}" aria-label="${e(t(snippets ? "片段" : "剪贴板历史"))}">${items}</ul>`;
}

export function showClipboardSnippetDialog(save: (name: string) => Promise<void>) {
  const dialog = document.createElement("dialog");
  dialog.className = "clipboard-snippet-dialog";
  dialog.setAttribute("aria-labelledby", "clipboard-snippet-title");
  dialog.innerHTML = `<form>
    <h2 id="clipboard-snippet-title">${e(t("另存为片段"))}</h2>
    <label>${e(t("片段名称"))}<input type="text" required autofocus autocomplete="off"></label>
    <p class="error" role="alert" hidden></p>
    <div class="clipboard-snippet-actions">
      <button type="button" class="secondary-button">${e(t("取消"))}</button>
      <button type="submit" class="primary-button">${e(t("保存片段"))}</button>
    </div>
  </form>`;
  const form = dialog.querySelector("form")!;
  const input = form.querySelector("input")!;
  const error = form.querySelector<HTMLElement>("[role=alert]")!;
  const cancel = form.querySelector<HTMLButtonElement>("[type=button]")!;
  const submit = form.querySelector<HTMLButtonElement>("[type=submit]")!;
  cancel.addEventListener("click", () => dialog.close());
  dialog.addEventListener("cancel", event => { if (submit.disabled) event.preventDefault(); });
  dialog.addEventListener("close", () => dialog.remove(), { once: true });
  input.addEventListener("input", () => { error.hidden = true; });
  form.addEventListener("submit", async event => {
    event.preventDefault();
    if (submit.disabled) return;
    const name = input.value.trim();
    if (!name || Array.from(name).length > 80) {
      error.textContent = t("片段名称须为 1 到 80 个字符。");
      error.hidden = false;
      input.focus();
      return;
    }
    submit.disabled = cancel.disabled = input.readOnly = true;
    error.hidden = true;
    try { await save(name); dialog.close(); }
    catch (reason) {
      error.textContent = `${t("无法保存片段：")}${String(reason)}`;
      error.hidden = false;
      input.focus();
    } finally { submit.disabled = cancel.disabled = input.readOnly = false; }
  });
  document.body.append(dialog);
  dialog.showModal();
}

export function showClipboardPreview(id: string) {
  const dialog = document.createElement("dialog");
  dialog.className = "clipboard-preview-dialog";
  dialog.setAttribute("aria-label", t("预览图片"));
  const close = document.createElement("button");
  close.type = "button";
  close.className = "icon-button";
  close.setAttribute("aria-label", t("关闭"));
  close.innerHTML = icon("close");
  close.addEventListener("click", () => dialog.close());
  const content = document.createElement("div");
  content.className = "clipboard-preview-content";
  content.tabIndex = -1;
  content.autofocus = true;
  const image = document.createElement("img");
  image.alt = t("图片");
  image.addEventListener("error", () => {
    const message = document.createElement("p");
    message.textContent = t("无法预览这张图片。");
    image.replaceWith(message);
  });
  content.append(close, image);
  dialog.append(content);
  dialog.addEventListener("close", () => dialog.remove(), { once: true });
  document.body.append(dialog);
  dialog.showModal();
  image.src = clipboardImageURL(id, true);
}
