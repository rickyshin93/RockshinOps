import { useState, type KeyboardEvent } from "react";
import { useTranslation } from "react-i18next";
import type { SettingsState } from "./settings";
import { Group, Row, Switch } from "./settings-controls";
import { acceleratorFromEvent, formatShortcut } from "./shortcut";

// Click, then press the new shortcut; Escape or leaving the control cancels.
function ShortcutRecorder({ value, isMacOS, disabled, labelledBy, onChange }: { value: string; isMacOS: boolean; disabled: boolean; labelledBy: string; onChange: (shortcut: string) => void }) {
  const { t } = useTranslation();
  const [recording, setRecording] = useState(false);
  const onKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
    if (!recording) return;
    event.preventDefault();
    event.stopPropagation();
    if (event.key === "Escape" && !event.metaKey && !event.ctrlKey && !event.altKey) { setRecording(false); return; }
    const shortcut = acceleratorFromEvent(event.nativeEvent);
    if (!shortcut) return;
    setRecording(false);
    if (shortcut !== value) onChange(shortcut);
  };
  // WebKit does not focus buttons on click; recording needs keyboard focus.
  return <button type="button" className={recording ? "shortcut-recorder recording" : "shortcut-recorder"} aria-labelledby={labelledBy} disabled={disabled} onClick={event => { event.currentTarget.focus(); setRecording(true); }} onKeyDown={onKeyDown} onBlur={() => setRecording(false)}>{recording ? t("按下快捷键…") : formatShortcut(value, isMacOS)}</button>;
}

export function LauncherSettingsView({ state, isMacOS, onLauncherShortcut }: { state: SettingsState; isMacOS: boolean; onLauncherShortcut: (shortcut: string) => void }) {
  const { t } = useTranslation();
  if (!state.launcher) return <div className="empty-state"><h2>{t("读取设置")}</h2><p>{state.launcherError || t("正在读取本机偏好…")}</p>{state.launcherError && <button type="button" className="secondary-button" data-action="load-launcher">{t("重试")}</button>}</div>;
  return <div className="settings-column launcher-settings-page">
    {(state.launcherError || state.launcher.error) && <div className="notice error" role="alert">{state.launcherError || state.launcher.error}</div>}
    <Group id="launcher-title" title={t("启用与快捷键")}>
      <Row id="launcher-label" label={t("启用启动器")} description={t("按快捷键在任意应用上方搜索并打开应用和文件。")}><Switch on={state.launcher.enabled} labelledBy="launcher-label" action="toggle-launcher" disabled={state.launcherBusy} /></Row>
      <Row id="launcher-shortcut-label" label={t("唤出快捷键")} description={t("点击后按下新的快捷键，按 Esc 取消。")}><ShortcutRecorder value={state.launcher.shortcut} isMacOS={isMacOS} disabled={state.launcherBusy} labelledBy="launcher-shortcut-label" onChange={onLauncherShortcut} /></Row>
      {state.launcher.enabled && state.launcher.spotlightConflict && <div className="notice warning" role="status">{t("聚焦搜索仍占用 ⌘ Space，按下时两者会同时弹出。请在“键盘快捷键 › 聚焦”中关闭“显示聚焦搜索”。")}<button type="button" className="secondary-button" data-action="open-shortcut-settings">{t("打开键盘设置")}</button></div>}
    </Group>
    {state.launcher.enabled && <Group id="launcher-search-title" title={t("搜索范围")}>
      <Row id="launcher-files-label" label={t("搜索文件")} description={isMacOS ? t("只索引个人文件夹中的文件名，不读取内容；包含 iCloud 云盘，跳过依赖、缓存和其他 App 的私有目录。") : t("只索引个人文件夹中的文件名，不读取内容；跳过依赖和缓存。")}><span className="setting-value">{state.launcher.files ? state.launcher.indexing ? t("正在建立索引…") : t("{{count}} 个项目", { count: state.launcher.fileCount.toLocaleString() }) : ""}</span><Switch on={state.launcher.files} labelledBy="launcher-files-label" action="toggle-launcher-files" disabled={state.launcherBusy} /></Row>
      {isMacOS && <Row id="launcher-wechat-label" label={t("搜索微信文件（仅本次运行）")} description={t("手动开启后索引微信收到的文件，macOS 可能请求访问其他 App 的数据。退出应用后自动关闭，下次启动不会扫描微信目录。")}><Switch on={state.launcher.weChat} labelledBy="launcher-wechat-label" action="toggle-launcher-wechat" disabled={state.launcherBusy || !state.launcher.files} /></Row>}
    </Group>}
  </div>;
}
