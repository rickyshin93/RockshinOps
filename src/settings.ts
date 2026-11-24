import type { LauncherSettings, MenuBarPreview, Preferences, StartupSettings } from "./mygo";
import type { Language } from "./i18n";

export const menuBarChoices = [
  ["disk", "磁盘剩余", "当前用户可写的剩余容量"],
  ["cpu", "CPU", "全部逻辑核心的总使用率"],
  ["memory", "内存", "已用内存占总内存的比例"],
  ["gpu", "GPU", "图形处理器使用率，不可用时显示 —"],
  ["network", "网络速度", "物理网络接口的下载与上传速度"],
  ["battery", "电池", "剩余电量，无电池时显示 —"],
] as const;

export interface SettingsState {
  startup: StartupSettings | null;
  startupBusy: boolean;
  startupError: string;
  launcher: LauncherSettings | null;
  launcherBusy: boolean;
  launcherError: string;
  preferences: Preferences | null;
  menuBarDraft: string[];
  trayPreview: MenuBarPreview | null;
  languageDraft: Language;
  updateProxyDraft: boolean;
  hideDockDraft: boolean;
  sidebarToolsDraft: Preferences["sidebarTools"];
  preferencesBusy: boolean;
  preferencesError: string;
}

export function preferencesDirty(state: SettingsState): boolean {
  const saved = state.preferences;
  return !!saved && (
    state.hideDockDraft !== saved.hideDockOnClose ||
    state.updateProxyDraft !== saved.updateUseSystemProxy ||
    state.languageDraft !== saved.language ||
    JSON.stringify(state.menuBarDraft) !== JSON.stringify(saved.menuBarItems) ||
    state.sidebarToolsDraft.calculator !== saved.sidebarTools.calculator ||
    state.sidebarToolsDraft.clipboard !== saved.sidebarTools.clipboard
  );
}
