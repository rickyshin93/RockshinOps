import { useMemo, useState, useSyncExternalStore } from "react";
import { useTranslation } from "react-i18next";
import { CleanupConfirmation } from "./cleanup-confirmation";
import { cleanupStore, restoreRecommended, revealCleanup, scanCleanup, selectCleanup, showCleanupMode, stopCleanup, trashSelectedCleanup } from "./cleanup-store";
import { formatBytes, formatCapacity } from "./format";
import { icon } from "./icons";
import type { CleanupItem, CleanupScan, ScanTarget, Volume } from "./mygo";
import { spaceStore } from "./space-store";
import { SpaceView } from "./space-view";

const Icon = ({ name }: { name: string }) => <span className="inline-flex" aria-hidden="true" dangerouslySetInnerHTML={{ __html: icon(name) }} />;
const useCleanup = () => useSyncExternalStore(cleanupStore.subscribe, cleanupStore.get, cleanupStore.get);
const categoryIcons: Record<string, string> = {
  "应用缓存": "folder", "应用日志": "snippet", "临时文件": "folder", "下载文件": "hide", "Xcode 构建缓存": "apps",
};
const sumBytes = (items: CleanupItem[]) => items.reduce((sum, item) => sum + item.bytes, 0);
const pageSize = 40;

type Props = {
  volume: Volume;
  capacityError: string;
  isMacOS: boolean;
  openDiskAccess: () => void;
  targets: ScanTarget[];
};

export function CleanupView(props: Props) {
  const { t } = useTranslation();
  const { mode, moving } = useCleanup();
  const space = useSyncExternalStore(spaceStore.subscribe, spaceStore.get, spaceStore.get);
  return <div className="cleanup-workspace">
    <div className="cleanup-modes">
      <div className="segmented" role="group" aria-label={t("清理方式")}>
        {([["quick", t("快速清理")], ["space", t("空间分析")]] as const).map(([id, label]) =>
          <button key={id} className={mode === id ? "selected" : ""} aria-pressed={mode === id}
            disabled={mode !== id && (moving || space.scanning || !!space.busy)} onClick={() => showCleanupMode(id)}>{label}</button>)}
      </div>
    </div>
    {mode === "quick" ? <QuickCleanup {...props} /> : <div className="cleanup-content"><SpaceView {...props} /></div>}
  </div>;
}

function QuickCleanup({ volume, capacityError, isMacOS, openDiskAccess, targets }: Props) {
  const { t, i18n } = useTranslation();
  const { scan, selected, scanning, moving, stopping, progress, outcome, error, cancelled } = useCleanup();
  const [confirming, setConfirming] = useState(false);
  const items = useMemo(() => scan?.categories.flatMap(category => category.items) ?? [], [scan]);
  const selection = items.filter(item => selected.has(item.path));
  const selectionSize = sumBytes(selection);
  const count = (value: number) => value.toLocaleString(i18n.language);

  return <>
    <div className="cleanup-content">
      {error && <div className="notice error" role="alert">{t("操作未完成：")}{error}</div>}
      {cancelled && <div className="notice" role="status">{scan ? t("扫描已停止，保留上一次的结果。") : t("扫描已停止。")}</div>}
      {scanning ? <section className="cleanup-start space-card" aria-labelledby="cleanup-scan-title">
        <span className="cleanup-symbol"><Icon name="scan" /></span>
        <div className="cleanup-start-copy">
          <h2 id="cleanup-scan-title">{stopping ? t("正在停止扫描…") : t("正在扫描清理位置")}</h2>
          <p role="status">{progress ? t(progress.category) : t("正在准备…")}</p>
          <progress className="cleanup-progress" aria-label={t("扫描进度")} max={progress?.total} value={progress?.completed} />
        </div>
        <button className="secondary-button" disabled={!progress || stopping} onClick={() => void stopCleanup()}>{t("停止扫描")}</button>
      </section> : !scan ? <>
        <section className="cleanup-start space-card" aria-labelledby="cleanup-start-title">
          <span className="cleanup-symbol"><Icon name="cleanup" /></span>
          <div className="cleanup-start-copy">
            <h2 id="cleanup-start-title">{t("看看哪些文件可以清理")}</h2>
            <p>{t("扫描常见清理位置，按占用大小审阅文件，再把所选项目移入废纸篓。")}</p>
          </div>
          <button className="primary-button" onClick={() => void scanCleanup()}><Icon name="scan" />{t("开始扫描")}</button>
        </section>
        <section className="space-card cleanup-scope">
          <h2>{t("扫描范围")}</h2>
          <ul>{targets.filter(target => target.cleanup).map(target => <li key={target.path}>
            <Icon name={categoryIcons[target.name] ?? "folder"} /><span>{t(target.name)}</span>
          </li>)}</ul>
          <div className="cleanup-capacity">
            <span>{t("剩余可写空间")}</span>
            <strong title={capacityError || volume.name}>{capacityError || volume.total <= 0 ? t("不可用") : formatCapacity(volume.available)}</strong>
          </div>
        </section>
      </> : <>
        <div className="cleanup-result-head">
          <div><h2>{items.length ? t("待审阅 {{size}}", { size: formatBytes(sumBytes(items)) }) : outcome ? t("本次清理结果") : t("没有找到可清理的项目")}</h2>
            <p>{items.length ? t("超过 {{days}} 天未更新的缓存、日志和临时文件已预选。", { days: count(scan.idleDays) }) : t("可以重新扫描，或用空间分析查看整块磁盘。")}</p></div>
          <button className="secondary-button" disabled={moving} onClick={() => void scanCleanup()}><Icon name="refresh" />{t("重新扫描")}</button>
        </div>
        {outcome && <div className={outcome.failed.length ? "notice warning" : "notice"} role="status">
          {outcome.moved.length > 0 && <p>{t("已将 {{count}} 项（{{size}}）移入废纸篓。清空后才会释放空间。", { count: count(outcome.moved.length), size: formatBytes(sumBytes(outcome.moved)) })}</p>}
          {outcome.failed.length > 0 && <details>
            <summary>{t("有 {{count}} 项未能移动", { count: count(outcome.failed.length) })}</summary>
            <ul className="cleanup-issues">{outcome.failed.map(item => <li key={item.path}><span className="path">{item.path}</span>
              <span>{item.changed ? t("文件已变化或无法读取。请重新扫描后审阅。") : item.error}</span></li>)}</ul>
          </details>}
        </div>}
        {scan.categories.some(category => category.error || category.skipped) && <div className="space-access">
          <details><summary>{t("部分位置无法纳入清理结果")}</summary>
            <ul className="cleanup-issues">{scan.categories.filter(category => category.error || category.skipped).map(category =>
              <li key={category.path}><strong>{t(category.name)}</strong><span>{category.error || t("已跳过 {{count}} 项", { count: count(category.skipped) })}</span></li>)}</ul>
          </details>
          {isMacOS && <button className="cleanup-action" onClick={openDiskAccess}>{t("打开完全磁盘访问权限")}</button>}
        </div>}
        {items.length > 0 && <CleanupReview key={scan.id} scan={scan} items={items} />}
      </>}
    </div>
    {!scanning && scan && items.length > 0 && <div className="cleanup-footer">
      <div aria-live="polite"><strong>{t("已选 {{count}} 项 · {{size}}", { count: count(selection.length), size: formatBytes(selectionSize) })}</strong>
        <p>{t("所选项目将移入废纸篓。")}</p></div>
      <button className="primary-button" aria-haspopup="dialog" disabled={!selection.length || moving} onClick={() => setConfirming(true)}>
        <Icon name="trash" />{moving ? t("正在移入废纸篓…") : t("移入废纸篓")}</button>
    </div>}
    {confirming && <CleanupConfirmation request={{
      message: t("将所选 {{count}} 项（{{size}}）移入废纸篓？所选文件夹的内容也会一起移动。", { count: count(selection.length), size: formatBytes(selectionSize) }),
      confirm: () => void trashSelectedCleanup(),
    }} onClose={() => setConfirming(false)} />}
  </>;
}

function CleanupReview({ scan, items }: { scan: CleanupScan; items: CleanupItem[] }) {
  const { t, i18n } = useTranslation();
  const { selected, moving } = useCleanup();
  const [categoryPath, setCategoryPath] = useState("");
  const [query, setQuery] = useState("");
  const [onlySelected, setOnlySelected] = useState(false);
  const [limit, setLimit] = useState(pageSize);
  const category = scan.categories.find(entry => entry.path === categoryPath);
  const sorted = useMemo(() => category?.items ?? [...items].sort((a, b) => b.bytes - a.bytes), [category, items]);
  const needle = query.trim().toLocaleLowerCase(i18n.language);
  const matching = sorted.filter(item => (!needle || item.path.toLocaleLowerCase(i18n.language).includes(needle)) && (!onlySelected || selected.has(item.path)));
  const scope = matching.map(item => item.path);
  const checked = matching.filter(item => selected.has(item.path)).length;
  const count = (value: number) => value.toLocaleString(i18n.language);

  return <section className="space-card cleanup-review" aria-label={t("审阅清理项目")}>
    <div className="cleanup-categories" role="group" aria-label={t("清理类别")}>
      {[{ name: "全部项目", path: "", items, error: "" }, ...scan.categories].map(entry =>
        <button key={entry.path} aria-pressed={categoryPath === entry.path} onClick={() => { setCategoryPath(entry.path); setLimit(pageSize); }}>
          <Icon name={categoryIcons[entry.name] ?? "list"} />
          <span>{t(entry.name)}</span><strong>{entry.error ? "—" : formatBytes(sumBytes(entry.items))}</strong>
        </button>)}
    </div>
    {category && (category.mode === "review" || category.mode === "downloads") && <p className="cleanup-manual">{t("此类别的项目由你手动选择。")}</p>}
    <div className="cleanup-filters">
      <label className="cleanup-search"><Icon name="search" />
        <input type="search" aria-label={t("搜索名称或路径")} placeholder={t("搜索名称或路径")} value={query}
          onChange={event => { setQuery(event.currentTarget.value); setLimit(pageSize); }} />
      </label>
      <label className="checkbox-label"><input type="checkbox" checked={onlySelected}
        onChange={event => { setOnlySelected(event.currentTarget.checked); setLimit(pageSize); }} />{t("仅看已选")}</label>
    </div>
    <div className="cleanup-list-head">
      <label className="checkbox-label"><input type="checkbox" disabled={!scope.length || moving} checked={checked > 0 && checked === scope.length}
        ref={element => { if (element) element.indeterminate = checked > 0 && checked < scope.length; }}
        onChange={event => selectCleanup(scope, event.currentTarget.checked)} /><span>{t("选择当前结果")}</span></label>
      <button className="cleanup-action" disabled={moving} onClick={restoreRecommended}>{t("恢复推荐")}</button>
    </div>
    {matching.length === 0 ? <p className="space-empty">{t("没有匹配的清理项目。")}</p> :
      <ul className="cleanup-list">{matching.slice(0, limit).map(item => <li key={item.path}>
        <label className="cleanup-file" title={item.path}>
          <input type="checkbox" aria-label={item.name} disabled={moving} checked={selected.has(item.path)} onChange={event => selectCleanup([item.path], event.currentTarget.checked)} />
          <span className="space-row-icon"><Icon name={item.directory ? "folder" : "file"} /></span>
          <span className="space-row-name"><strong>{item.name}</strong><small className="path">{item.path}</small></span>
          <span className="cleanup-file-size">{formatBytes(item.bytes)}</span>
        </label>
        <button className="space-reveal" disabled={moving} onClick={() => void revealCleanup(item.path)} title={t("在文件管理器中显示")}
          aria-label={t("在文件管理器中显示「{{name}}」", { name: item.name })}><Icon name="reveal" /></button>
      </li>)}</ul>}
    {matching.length > limit && <div className="cleanup-more">
      <span>{t("已显示 {{shown}} / {{total}} 项", { shown: count(limit), total: count(matching.length) })}</span>
      <button className="secondary-button" onClick={() => setLimit(limit + pageSize)}>{t("显示更多")}</button>
    </div>}
  </section>;
}
