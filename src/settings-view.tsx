import { useTranslation } from "react-i18next";
import { icon } from "./icons";
import { menuBarChoices, preferencesDirty, type SettingsState } from "./settings";
import type { Language } from "./i18n";

export function SettingsView({ state, onLanguage }: { state: SettingsState; onLanguage: (language: Language) => void }) {
  const { t } = useTranslation();
  if (!state.preferences) return <div className="empty-state"><h2>{t("读取设置")}</h2><p>{state.preferencesError || t("正在读取本机偏好…")}</p>{state.preferencesError && <button className="secondary-button" data-action="load-settings">{t("重试")}</button>}</div>;
  return <div className="settings-page">
    <section className="settings-options language-settings" aria-labelledby="language-title">
      <div className="setting-copy"><h2 id="language-title">{t("显示语言")}</h2><p>{t("默认跟随系统语言；不支持的语言使用英文。")}</p></div>
      <div className="language-choices" role="radiogroup" aria-labelledby="language-title">
        {([['system', t('跟随系统')], ['zh', '中文'], ['en', 'English']] as const).map(([id, label]) => <label key={id}><input type="radio" name="language" value={id} checked={state.languageDraft === id} disabled={state.preferencesBusy} onChange={() => onLanguage(id)} /><span>{label}</span></label>)}
      </div>
    </section>
    <section className="settings-preview" aria-label={t("菜单栏显示项目预览")}><div className="preview-heading"><h2>{t("菜单栏")}</h2><span>{t("屏幕顶部 · LocalOps")}</span></div><div className="tray-preview"><span dangerouslySetInnerHTML={{ __html: icon("disk") }} />{state.menuBarDraft.length ? state.menuBarDraft.map(id => <span key={id}>{t(menuBarChoices.find(([key]) => key === id)![1])}</span>) : <span>{t("只显示图标")}</span>}</div><p>{t("选择需要常驻的指标。保存后立即生效，关闭窗口也会持续更新。")}</p></section>
    <section className="settings-options" aria-label={t("菜单栏指标")}>{menuBarChoices.map(([id, title, description]) => <div className="setting-row" key={id}><span className={`setting-icon ${id === 'battery' ? 'energy' : id}`} dangerouslySetInnerHTML={{ __html: icon(id) }} /><div className="setting-copy"><label id={`setting-label-${id}`}>{t(title)}</label><p>{t(description)}</p></div><button type="button" className="switch" role="switch" aria-checked={state.menuBarDraft.includes(id)} aria-labelledby={`setting-label-${id}`} data-metric={id} disabled={state.preferencesBusy}><span /></button></div>)}</section>
    <div className="settings-save"><p role="status">{t(state.preferencesBusy ? "正在保存…" : preferencesDirty(state) ? "有未保存的更改" : "已保存到本机")}</p><button className="primary-button" data-action="save-settings" disabled={!preferencesDirty(state) || state.preferencesBusy}>{t("保存设置")}</button></div>
    {state.preferencesError && <div className="notice error" role="alert">{state.preferencesError}</div>}
    <p className="source-note">{t("全部关闭时仅保留 LocalOps 图标。显示项目越多，占用的菜单栏空间越宽。")}</p>
  </div>;
}
