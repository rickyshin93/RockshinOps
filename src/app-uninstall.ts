import { toggleAllAppFiles, reconcileAppFiles } from "./app-selection";
import { escapeHTML as e } from "./format";
import { appIcon, icon } from "./icons";
import { t } from "./i18n";
import { System, type AppReview, type FileOutcome, type InstalledApp } from "./mygo";
import { invalidateSpace } from "./space-store";

const state = {
  installed: false, apps: null as InstalledApp[] | null, review: null as AppReview | null,
  selection: [] as string[], busy: false, searching: false, message: "", outcome: null as FileOutcome | null,
};
let reviewEpoch = 0;
let selectAll = false;
let notify = (_filters = false) => {};
export function appUninstallStatus() { return { installed: state.installed, reviewing: state.installed && state.review !== null }; }

async function showInstalled() {
  if (state.busy) return;
  Object.assign(state, { installed: true, review: null, selection: [], busy: true, message: "", outcome: null });
  notify(true);
  try { state.apps = await System.installedApps(); }
  catch (error) { state.apps = null; state.message = String(error); }
  finally { state.busy = false; notify(); }
}

async function reviewApp(path: string) {
  if (state.busy) return;
  const epoch = ++reviewEpoch;
  selectAll = false;
  Object.assign(state, { review: null, selection: [], busy: true, searching: true, message: "", outcome: null });
  notify(true);
  try {
    const quick = await System.reviewAppQuick(path);
    if (epoch !== reviewEpoch) { await System.cancelAppReview(quick.reviewId); return; }
    state.review = quick;
    notify(true);
    const review = await System.reviewApp(path, quick.reviewId);
    if (epoch !== reviewEpoch) return;
    state.review = review;
    state.selection = reconcileAppFiles(review.files.map(file => file.path), state.selection, selectAll);
  } catch (error) {
    if (epoch === reviewEpoch) state.message = String(error);
  } finally {
    if (epoch === reviewEpoch) {
      state.busy = false;
      state.searching = false;
      notify(true);
    }
  }
}

async function stopSearch() {
  if (!state.searching || !state.review) return;
  const epoch = ++reviewEpoch;
  try {
    await System.cancelAppReview(state.review.reviewId);
    if (epoch === reviewEpoch) state.message = t("已停止搜索，仅保留常见位置的首批项目。");
  } catch (error) {
    if (epoch === reviewEpoch) state.message = String(error);
  } finally {
    if (epoch === reviewEpoch) {
      state.busy = false;
      state.searching = false;
      notify();
    }
  }
}

function showRunning() {
  if (state.searching && state.review) void stopSearch();
  else if (state.searching) { reviewEpoch++; state.busy = false; state.searching = false; }
  state.installed = false;
  state.review = null;
  notify(true);
}

async function removeFiles() {
  const { review, selection } = state;
  if (state.busy || !review || !selection.length) return;
  Object.assign(state, { busy: true, message: "", outcome: null });
  notify();
  try {
    const outcome = await System.removeAppFiles(review.reviewId, selection);
    state.outcome = outcome;
    if (outcome.succeeded.length > 0) {
      invalidateSpace();
      state.message = t("已将 {{count}} 项移入废纸篓。", { count: outcome.succeeded.length });
    }
    const removed = new Set([...outcome.succeeded, ...outcome.failed.filter(item => item.changed).map(item => item.path)]);
    state.selection = selection.filter(path => !removed.has(path));
    state.review = outcome.failed.length ? { ...review, files: review.files.filter(file => !removed.has(file.path)) } : null;
    if (!state.review) state.apps = await System.installedApps();
  } catch (error) { state.message = String(error); }
  finally { state.busy = false; notify(true); }
}

export function bindAppUninstall(root: HTMLElement, render: (filters?: boolean) => void) {
  notify = render;
  root.addEventListener("click", event => {
    const button = (event.target as Element).closest<HTMLButtonElement>("button");
    if (!button || button.disabled) return;
    if (button.dataset.reviewApp) { void reviewApp(button.dataset.reviewApp); return; }
    switch (button.dataset.action) {
      case "running-apps": showRunning(); break;
      case "installed-apps": void showInstalled(); break;
      case "stop-app-search": void stopSearch(); break;
      case "remove-app-files": root.querySelector<HTMLDialogElement>(".uninstall-confirm")?.showModal(); break;
    }
  });
  root.addEventListener("submit", event => {
    if ((event as SubmitEvent).submitter?.hasAttribute("data-confirm-uninstall")) void removeFiles();
  });
  root.addEventListener("change", event => {
    const input = event.target as HTMLInputElement;
    if (!state.review || !(input.dataset.appSelectAll || input.dataset.appFile)) return;
    if (input.dataset.appSelectAll) {
      state.selection = toggleAllAppFiles(state.review.files.map(file => file.path), !input.checked);
    } else {
      const selected = new Set(state.selection);
      if (input.checked) selected.add(input.dataset.appFile!);
      else selected.delete(input.dataset.appFile!);
      state.selection = [...selected];
    }
    selectAll = state.review.files.length > 0 && state.selection.length === state.review.files.length;
    notify();
  });
}

export function syncAppSelection(root: HTMLElement) {
  const checkbox = root.querySelector<HTMLInputElement>("[data-app-select-all]");
  if (checkbox && state.review) checkbox.indeterminate = state.selection.length > 0 && state.selection.length < state.review.files.length;
}

function empty(title: string, detail: string): string {
  return `<div class="empty-state"><h2>${e(title)}</h2><p>${e(detail)}</p></div>`;
}

function appScanCoverage(review: AppReview): string {
  if (!review.scanIssues.length) return "";
  const issueLabel = (kind: string) => kind === "permission" ? t("没有访问权限") : kind === "changed" ? t("扫描期间位置已变化") : t("读取失败");
  const permission = review.scanIssues.some(issue => issue.kind === "permission") ? `<div class="disk-access-prompt" role="status"><div><strong>${t("部分位置需要访问权限")}</strong><p>${t("开启完全磁盘访问权限可扩大搜索范围。在系统设置中允许 RockshinOps；若列表里没有，点“+”添加。开启后退出并重新打开 RockshinOps，再重新搜索。部分系统目录仍可能受限。")}</p></div><button class="secondary-button" data-action="open-full-disk-access">${t("打开完全磁盘访问权限")}</button></div>` : "";
  return permission + `<details class="uninstall-coverage"><summary><strong>${t("扫描覆盖范围受限")}</strong><span>${t("未读取的位置：{{count}} 个", { count: review.scanIssues.length })}</span></summary><p>${t("这是整个搜索范围的访问情况，不是此 App 的残留数量。已找到的项目仍可核对清理；未读取的位置中可能还有关联文件。")}</p><ul>${review.scanIssues.map(issue => `<li><span class="path">${e(issue.path)}</span><small>${e(issueLabel(issue.kind))} · ${e(issue.error)}</small></li>`).join("")}</ul></details>`;
}

export function installedAppsView(query: string, view: Readonly<Omit<typeof state, "installed">> = state): string {
  const message = view.message ? `<div class="notice" role="status">${e(view.message)}</div>` : "";
  const outcome = view.outcome?.failed.length ? `<details class="notice warning uninstall-coverage"><summary>${t("有 {{count}} 项未能移动", { count: view.outcome.failed.length })}</summary><ul>${view.outcome.failed.map(item => `<li><span class="path">${e(item.path)}</span><small>${e(item.changed ? t("文件已变化或无法读取。请重新扫描后审阅。") : item.error)}</small></li>`).join("")}</ul></details>` : "";
  if (view.busy && !view.review) return message + outcome + empty(t("正在读取 App"), t("请稍候。"));
  if (view.review) {
    const review = view.review;
    const selected = view.selection.length;
    const allSelected = review.files.length > 0 && selected === review.files.length;
    return message + outcome + `<button class="cleanup-action uninstall-back" data-action="installed-apps" ${view.busy ? "disabled" : ""}>${t("返回应用列表")}</button><section class="uninstall-review"><div class="uninstall-heading">${appIcon(review.app.path, true)}<div><h2>${e(review.app.name)}</h2><p class="path">${e(review.bundleId || review.app.path)}</p></div></div><p class="uninstall-guidance">${t("请逐项核对后选择。名称匹配可能属于共享数据；搜索无法保证找到所有残留。移除前请退出 App。")}</p>${view.searching ? `<div class="uninstall-progress" role="status"><span class="loading-dot" aria-hidden="true"></span><span>${t("正在继续搜索关联文件…")}</span><button class="cleanup-action" data-action="stop-app-search">${t("停止搜索")}</button></div>` : ""}${appScanCoverage(review)}<div class="uninstall-list-heading"><label><input type="checkbox" data-app-select-all="true" aria-label="${t("全选关联项目")}" ${allSelected ? "checked" : ""} ${view.busy && !view.searching ? "disabled" : ""}><span class="uninstall-list-title">${t("关联项目")}</span></label><span>${t("已选择 {{selected}} / {{total}} 项", { selected, total: review.files.length })}</span></div><div class="uninstall-list">${review.files.map(file => `<label><input type="checkbox" data-app-file="${e(file.path)}" ${view.busy && !view.searching ? "disabled" : ""} ${view.selection.includes(file.path) ? "checked" : ""}><span><strong class="path">${e(file.path)}</strong><small>${e(t(file.reason))}</small></span></label>`).join("")}</div><div class="uninstall-actions"><span>${view.searching ? t("搜索完成或停止后可移入废纸篓。") : t("所选项目将移入废纸篓。")}</span><button class="primary-button" data-action="remove-app-files" aria-haspopup="dialog" ${selected && !view.busy ? "" : "disabled"}>${t("移入废纸篓")}</button></div>
      <dialog class="cleanup-dialog uninstall-confirm" aria-labelledby="uninstall-confirm-title" aria-describedby="uninstall-confirm-detail">
        <h2 id="uninstall-confirm-title">${t("确认移入废纸篓")}</h2>
        <p id="uninstall-confirm-detail">${t("将以下选中项目移入系统废纸篓？请确认 App 已退出。")}</p>
        <ul class="cleanup-issues">${view.selection.map(path => `<li class="path">${e(path)}</li>`).join("")}</ul>
        <form method="dialog"><button class="secondary-button" autofocus>${t("取消")}</button><button class="primary-button" data-confirm-uninstall>${t("确认移动")}</button></form>
      </dialog></section>`;
  }
  if (!view.apps) return message + outcome + empty(t("无法读取已安装的 App"), t("请重试。"));
  const apps = view.apps.filter(app => `${app.name} ${app.path}`.toLowerCase().includes(query.toLowerCase()));
  return message + outcome + `<div class="installed-hint"><span>${t("选择 App 后可查看应用包和可能的关联文件。")}</span><button class="cleanup-action" data-action="open-full-disk-access">${t("打开完全磁盘访问权限")}</button></div>` + (apps.length ? `<div class="app-list installed-list">${apps.map(app => `<button data-review-app="${e(app.path)}">${appIcon(app.path, true)}<span><strong>${e(app.name)}</strong><small class="path">${e(app.path)}</small></span>${icon("chevron")}</button>`).join("")}</div>` : empty(t("没有匹配的 App"), t("试试其他名称或路径。")));
}
