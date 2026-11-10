import { useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { icon } from "./icons";
import type { CalculatorFormat } from "./mygo";

const Icon = ({ name }: { name: string }) => <span className="inline-flex" aria-hidden="true" dangerouslySetInnerHTML={{ __html: icon(name) }} />;

export function CalculatorSettings({
  format, maxDecimals, hasHistory, busy, actions, onFormat, onClear,
}: {
  format?: CalculatorFormat;
  maxDecimals: number;
  hasHistory: boolean;
  busy: boolean;
  actions?: ReactNode;
  onFormat: (format: CalculatorFormat) => Promise<void>;
  onClear: () => Promise<void>;
}) {
  const { t } = useTranslation();
  const dialog = useRef<HTMLDialogElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const [confirm, setConfirm] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const tools = document.getElementById("page-actions");
  const groupings = [["auto", t("跟随语言")], ["myriads", t("万位")], ["thousands", t("千位")], ["none", t("不分组")]] as const;

  async function run(action: () => Promise<void>, errorKey: string) {
    if (pending) return;
    setPending(true);
    setError("");
    try {
      await action();
      setConfirm(false);
    } catch (reason) {
      console.error(reason);
      setError(errorKey);
    } finally {
      setPending(false);
    }
  }

  return <>
    {tools && createPortal(<>{actions}<button ref={trigger} type="button" className="icon-button" disabled={busy}
      aria-label={t("计算器设置")} title={t("计算器设置")} aria-haspopup="dialog" onClick={() => dialog.current?.showModal()}><Icon name="settings" /></button></>, tools)}
    <dialog className="calculator-settings-dialog" ref={dialog} aria-labelledby="calculator-settings-title" aria-busy={pending} onKeyDown={event => event.stopPropagation()}
      onCancel={event => { if (pending) event.preventDefault(); }}
      onClose={() => { setConfirm(false); setError(""); trigger.current?.focus(); }}>
      <header>
        <h2 id="calculator-settings-title">{t("计算器设置")}</h2>
        <button type="button" className="icon-button" aria-label={t("关闭")} disabled={pending} onClick={() => dialog.current?.close()}><Icon name="close" /></button>
      </header>
      {format && <div className="calculator-format-settings">
        <div className="calculator-format-row">
          <span id="calculator-grouping-label">{t("数字分组")}</span>
          <div className="segmented" role="radiogroup" aria-labelledby="calculator-grouping-label">
            {groupings.map(([grouping, label]) => <label key={grouping}><input type="radio" name="calculator-grouping"
              checked={format.grouping === grouping} disabled={pending}
              onChange={() => void run(() => onFormat({ ...format, grouping }), "无法保存计算器设置。")} /><span>{label}</span></label>)}
          </div>
        </div>
        <div className="calculator-format-row">
          <span>{t("最多小数位")}</span>
          <div className="calculator-precision">
            <button type="button" className="icon-button" disabled={pending || format.decimalPlaces === 0} aria-label={t("减少小数位")}
              onClick={() => void run(() => onFormat({ ...format, decimalPlaces: format.decimalPlaces - 1 }), "无法保存计算器设置。")}>−</button>
            <output aria-label={t("最多小数位")}>{format.decimalPlaces}</output>
            <button type="button" className="icon-button" disabled={pending || format.decimalPlaces === maxDecimals} aria-label={t("增加小数位")}
              onClick={() => void run(() => onFormat({ ...format, decimalPlaces: format.decimalPlaces + 1 }), "无法保存计算器设置。")}>+</button>
          </div>
        </div>
      </div>}
      <section className="calculator-settings-history" aria-label={t("计算历史")}>
        {confirm ? <>
          <p>{t("清空全部计算历史？此操作无法撤销。")}</p>
          <div className="calculator-settings-confirm">
            <button type="button" className="secondary-button" autoFocus disabled={pending} onClick={() => setConfirm(false)}>{t("取消")}</button>
            <button type="button" className="secondary-button calculator-clear" disabled={pending} onClick={() => void run(onClear, "无法清空计算历史，请重试。")}>{t("确认清空")}</button>
          </div>
        </> : <button type="button" className="secondary-button calculator-clear" disabled={!hasHistory || pending} onClick={() => setConfirm(true)}>
          <Icon name="trash" />{t("清空计算历史")}
        </button>}
      </section>
      {error && <p className="calculator-settings-error" role="alert">{t(error)}</p>}
    </dialog>
  </>;
}
