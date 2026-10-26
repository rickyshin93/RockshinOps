import type { Preferences } from "./mygo";
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
  preferences: Preferences | null;
  menuBarDraft: string[];
  languageDraft: Language;
  preferencesBusy: boolean;
  preferencesError: string;
}

export function preferencesDirty(state: SettingsState): boolean {
  return !!state.preferences && (state.languageDraft !== state.preferences.language || JSON.stringify(state.menuBarDraft) !== JSON.stringify(state.preferences.menuBarItems));
}
