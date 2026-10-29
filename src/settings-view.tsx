import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { icon } from "./icons";
import { menuBarChoices, type SettingsState } from "./settings";
import type { Language } from "./i18n";

function Group({ id, title, note, children }: { id: string; title: string; note?: string; children: ReactNode }) {
  return <section className="settings-group" aria-labelledby={id}><h2 id={id}>{title}</h2><div className="settings-card">{children}</div>{note && <p className="settings-note">{note}</p>}</section>;
}

function Row({ id, label, description, lead, children }: { id: string; label: string; description?: string; lead?: ReactNode; children: ReactNode }) {
  return <div className={lead ? "settings-row with-lead" : "settings-row"}>{lead}<div className="setting-copy"><span id={id} className="setting-label">{label}</span>{description && <p>{description}</p>}</div><div className="setting-control">{children}</div></div>;
}

function Switch({ on, labelledBy, metric, action }: { on: boolean; labelledBy: string; metric?: string; action?: string }) {
  return <button type="button" className="switch" role="switch" aria-checked={on} aria-labelledby={labelledBy} data-metric={metric} data-action={action}><span /></button>;
}

export function SettingsView({ state, version, onLanguage, isMacOS }: { state: SettingsState; version: string; onLanguage: (language: Language) => void; isMacOS: boolean }) {
  const { t } = useTranslation();
  if (!state.preferences) return <div className="empty-state"><h2>{t("读取设置")}</h2><p>{state.preferencesError || t("正在读取本机偏好…")}</p>{state.preferencesError && <button className="secondary-button" data-action="load-settings">{t("重试")}</button>}</div>;
  const languages = [["system", t("跟随系统")], ["zh", "中文"], ["en", "English"]] as const;
  return <div className="settings-page">
    {state.preferencesError && <div className="notice error" role="alert">{state.preferencesError}</div>}
    <div className="settings-column">
    <Group id="menu-bar-title" title={t("菜单栏")} note={`${t("关闭窗口后菜单栏仍会持续更新。")}${t("全部关闭时仅保留 RockshinOps 图标。显示项目越多，占用的菜单栏空间越宽。")}`}>
      <div className="tray-preview" aria-hidden="true">{state.menuBarDraft.length && state.trayPreview?.image ? <img src={state.trayPreview.image} alt="" /> : <><span className="tray-icon" dangerouslySetInnerHTML={{ __html: icon("disk") }} />{state.menuBarDraft.length ? <span className="tray-text">{state.trayPreview?.text}</span> : <span className="tray-empty">{t("只显示图标")}</span>}</>}</div>
      {menuBarChoices.map(([id, title, description]) => <Row key={id} id={`setting-label-${id}`} label={t(title)} description={t(description)} lead={<span className={`content-icon ${id === "battery" ? "energy" : id}`} dangerouslySetInnerHTML={{ __html: icon(id) }} />}><Switch on={state.menuBarDraft.includes(id)} labelledBy={`setting-label-${id}`} metric={id} /></Row>)}
    </Group>
    </div>
    <div className="settings-column">
    <Group id="general-title" title={t("通用")}>
      {isMacOS && <Row id="hide-dock-label" label={t("关闭时隐藏程序坞图标")} description={t("关闭窗口时隐藏 RockshinOps 的程序坞图标，菜单栏仍会保留。重新打开窗口时恢复图标。")}><Switch on={state.hideDockDraft} labelledBy="hide-dock-label" action="toggle-hide-dock" /></Row>}
      <Row id="language-label" label={t("显示语言")} description={t("默认跟随系统语言；不支持的语言使用英文。")}>
        <div className="segmented" role="radiogroup" aria-labelledby="language-label">{languages.map(([id, label]) => <label key={id}><input type="radio" name="language" value={id} checked={state.languageDraft === id} onChange={() => onLanguage(id)} /><span>{label}</span></label>)}</div>
      </Row>
    </Group>
    <Group id="updates-title" title={t("软件更新")}>
      <Row id="version-label" label={t("版本")} description={t("启动后自动检查更新，默认确认后安装。系统软件包请通过包管理器更新。")}><span className="setting-value">{version || "—"}</span><button type="button" className="secondary-button" data-action="check-updates">{t("检查更新…")}</button></Row>
      {isMacOS && <Row id="update-proxy-label" label={t("更新时使用系统代理")} description={t("使用 macOS 网络设置中的代理连接更新服务器。")}><Switch on={state.updateProxyDraft} labelledBy="update-proxy-label" action="toggle-update-proxy" /></Row>}
    </Group>
    </div>
  </div>;
}
