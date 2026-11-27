import { createContext, useContext, useState, useSyncExternalStore, type CSSProperties } from "react";
import type { TFunction } from "i18next";
import { useTranslation } from "react-i18next";
import { CleanupConfirmation, type CleanupConfirmationRequest } from "./cleanup-confirmation";
import { icon } from "./icons";
import { formatBytes, formatCapacity, percent, timeLabel } from "./format";
import { System, type CleanupFailure, type SpaceEntry, type SpaceSummary, type TrashReview, type Volume } from "./mygo";
import { discardTrashReview, emptyTrash, openFolder, prepareTrash, scanSpace, showTab, spaceStore, stopScan, trashItem, type SpaceNotice, type SpaceTab } from "./space-store";

const Icon = ({ name }: { name: string }) => <span className="inline-flex" aria-hidden="true" dangerouslySetInnerHTML={{ __html: icon(name) }} />;
const Tile = ({ name }: { name: string }) => <span className="content-icon" aria-hidden="true" dangerouslySetInnerHTML={{ __html: icon(name) }} />;
const useSpace = () => useSyncExternalStore(spaceStore.subscribe, spaceStore.get, spaceStore.get);
const share = (part: number, whole: number) => whole > 0 ? Math.min(1, Math.max(0, part / whole)) : 0;
const SpaceConfirmation = createContext<((request: CleanupConfirmationRequest) => void) | null>(null);

export function SpaceView({ volume, capacityError, isMacOS, openDiskAccess }: { volume: Volume; capacityError: string; isMacOS: boolean; openDiskAccess: () => void }) {
  const { t } = useTranslation();
  const space = useSpace();
  const [confirmation, setConfirmation] = useState<CleanupConfirmationRequest | null>(null);
  const used = capacityError || volume.total <= 0 ? 0 : volume.total - volume.available;
  return <div className="space-page">
    <StorageCard volume={volume} capacityError={capacityError} summary={space.summary} />
    {space.notice && <Notice notice={space.notice} />}
    {space.listingError && <div className="notice error" role="alert">{t("无法读取文件夹：")}{space.listingError}</div>}
    {space.scanning ? <ScanCard used={used} /> : !space.summary ? <section className="space-card space-intro" aria-labelledby="space-intro-title">
      <div className="space-card-title"><Tile name="scan" /><h2 id="space-intro-title">{t("空间分析")}</h2></div>
      <p>{t("扫描整块磁盘，按实际占用找出最大的文件夹和文件，再逐层打开查看。")}</p>
      <ul className="space-facts"><li>{t("只读取文件大小，不读取内容")}</li><li>{t("移入废纸篓前会再次确认")}</li><li>{t("文件较多时需要一两分钟")}</li></ul>
      <div className="space-intro-actions"><button className="primary-button" onClick={() => void scanSpace()}><Icon name="scan" />{t("开始扫描")}</button>{isMacOS && <button className="secondary-button" onClick={openDiskAccess}>{t("打开完全磁盘访问权限")}</button>}</div>
    </section> : <SpaceConfirmation value={setConfirmation}><Results summary={space.summary} isMacOS={isMacOS} openDiskAccess={openDiskAccess} volumeName={volume.name || volume.mount} /></SpaceConfirmation>}
    {confirmation && <CleanupConfirmation request={confirmation} onClose={() => setConfirmation(null)} />}
  </div>;
}

function StorageCard({ volume, capacityError, summary }: { volume: Volume; capacityError: string; summary: SpaceSummary | null }) {
  const { t } = useTranslation();
  const capacity = !capacityError && volume.total > 0;
  const used = capacity ? volume.total - volume.available : 0;
  const named = (summary?.top ?? []).filter(entry => !entry.other).slice(0, 4);
  const scannedOther = summary ? Math.max(0, summary.bytes - named.reduce((sum, entry) => sum + entry.bytes, 0)) : 0;
  const segments: { label: string; bytes: number; tone: string }[] = summary
    ? [...named.map((entry, index) => ({ label: entry.name, bytes: entry.bytes, tone: `s${index}` })), { label: t("其他文件"), bytes: scannedOther, tone: "s4" }, { label: t("系统数据与未统计"), bytes: Math.max(0, used - summary.bytes), tone: "system" }]
    : [{ label: t("已占用及保留"), bytes: used, tone: "used" }];
  return <section className="storage-summary" aria-labelledby="storage-title">
    <div className="space-card-title"><Tile name="disk" /><h2 id="storage-title">{t("磁盘空间")}</h2><span className="ml-auto path text-muted">{volume.name || volume.mount}</span></div>
    {capacityError && <p className="mt-3 text-small text-error" role="status">{capacityError}</p>}
    <dl className="storage-values">
      <div><dt>{t("剩余可写空间")}</dt><dd className="storage-available">{capacity ? formatCapacity(volume.available) : t("不可用")}</dd></div>
      <div><dt>{t("已占用及保留")}</dt><dd>{capacity ? formatCapacity(used) : "—"}</dd></div>
      <div><dt>{t("容量")}</dt><dd>{capacity ? formatCapacity(volume.total) : "—"}</dd></div>
    </dl>
    {capacity && <>
      <div className="space-bar" role="img" aria-label={segments.map(segment => `${segment.label} ${formatCapacity(segment.bytes)}`).join(", ")}>
        {segments.map(segment => <i key={segment.label} data-tone={segment.tone} style={{ flexGrow: segment.bytes } as CSSProperties} />)}
        <i data-tone="free" style={{ flexGrow: volume.available }} />
      </div>
      <ul className="space-legend">{[...segments, { label: t("可用"), bytes: volume.available, tone: "free" }].filter(segment => segment.bytes > 0).map(segment => <li key={segment.label}><i data-tone={segment.tone} /><span>{segment.label}</span><strong>{formatCapacity(segment.bytes)}</strong></li>)}</ul>
    </>}
    <p className="mt-3 text-caption text-muted">{t("当前用户可写的剩余容量")} · {t("用户主目录所在卷")}{summary ? ` · ${t("按实际占用统计；APFS 克隆和快照可能导致与系统数字略有出入")}` : ""}</p>
  </section>;
}

function Notice({ notice }: { notice: SpaceNotice }) {
  const { t } = useTranslation();
  if (notice.kind === "purged") return <div className={notice.failed.length || notice.postDeleteFailures.length ? "notice warning" : "notice"} role="status">
    {notice.count > 0 && <p>{t("已永久删除 {{count}} 个文件或目录（{{size}}）。", { count: notice.count, size: formatBytes(notice.bytes) })}</p>}
    <TrashIssues title={t("有 {{count}} 个顶层项目未完整删除", { count: notice.failed.length })} issues={notice.failed} />
    <TrashIssues title={t("有 {{count}} 项删除后的清理步骤失败", { count: notice.postDeleteFailures.length })} issues={notice.postDeleteFailures} />
  </div>;
  const text = notice.kind === "trashed" ? t("已将「{{name}}」（{{size}}）移入废纸篓。清空废纸篓后才会释放空间。", { name: notice.name, size: formatCapacity(notice.bytes) })
    : notice.kind === "trashEmpty" ? t("废纸篓没有可删除的项目。")
    : notice.kind === "cancelled" ? t("扫描已停止，保留上一次的结果。")
    : `${t("操作未完成：")}${trashError(notice.message, t)}`;
  return <div className={notice.kind === "error" ? "notice error" : "notice"} role={notice.kind === "error" ? "alert" : "status"}>{text}</div>;
}

function trashError(message: string, t: TFunction) {
  if (message === "trash_review_changed") return t("废纸篓内容已变化，请重新审阅。");
  if (message === "trash_recovery_pending") return t("请查看保留在暂存位置的项目，再重新审阅。");
  if (message === "trash_metadata_changed") return t("恢复信息已变化，已保留现有信息。");
  if (message === "trash_metadata_invalid") return t("恢复信息不是可读取的普通文件，已保留。");
  return message;
}

function TrashIssues({ title, issues }: { title: string; issues: CleanupFailure[] }) {
  const { t } = useTranslation();
  if (!issues.length) return null;
  return <details open={issues.some(item => !!item.recoveryPath)}>
    <summary>{title}</summary>
    <ul className="cleanup-issues">{issues.map(item => <li key={item.path}>
      <span className="path">{item.path}</span>
      <span>{trashError(item.error, t)}</span>
      {item.recoveryPath && <span>{t("项目保留在暂存位置 {{path}}", { path: item.recoveryPath })}</span>}
    </li>)}</ul>
  </details>;
}

function TrashManifest({ review }: { review: TrashReview }) {
  const { t, i18n } = useTranslation();
  const [page, setPage] = useState({ offset: 0, items: review.items });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const shown = page.offset + page.items.length;
  const number = (value: number) => value.toLocaleString(i18n.language);

  async function load(offset: number) {
    setBusy(true);
    setError("");
    try {
      const items = offset === 0 ? review.items : await System.trashReviewItems(review.id, offset);
      setPage({ offset, items });
    } catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); }
    finally { setBusy(false); }
  }

  return <section className="trash-manifest" aria-label={t("本次复核清单")} aria-busy={busy}>
    <h3>{t("本次复核清单")}</h3>
    <p>{t("清单显示每项占用，预计释放空间已合并共享的硬链接。")}</p>
    <ul>{page.items.map(item => <li key={item.name}><span className="path">{item.name}</span><span>{formatBytes(item.bytes)}</span></li>)}</ul>
    {review.count > review.items.length && <div className="trash-manifest-controls">
      <span aria-live="polite">{t("第 {{first}}–{{last}} 项，共 {{total}} 项", { first: number(page.offset + 1), last: number(shown), total: number(review.count) })}</span>
      <div>
        <button className="secondary-button" disabled={busy || page.offset === 0} onClick={() => void load(page.offset - review.items.length)}>{t("上一页")}</button>
        <button className="secondary-button" disabled={busy || shown >= review.count} onClick={() => void load(shown)}>{t("下一页")}</button>
      </div>
    </div>}
    {error && <p role="alert">{trashError(error, t)}</p>}
  </section>;
}

function ScanCard({ used }: { used: number }) {
  const { t, i18n } = useTranslation();
  const { progress, progressFailed, stopping } = useSpace();
  const ratio = progress && used > 0 ? Math.min(99, Math.floor(progress.bytes / used * 100)) : null;
  return <section className="scan-card" role="status" aria-live="polite">
    <div className="scan-ring" data-indeterminate={ratio === null} style={{ "--scan-progress": ratio ?? 0 } as CSSProperties}>
      <span className="scan-ring-value">{ratio === null ? <Tile name="cleanup" /> : `${ratio}%`}</span>
    </div>
    <h2>{stopping ? t("正在停止扫描…") : t("正在扫描本机空间")}</h2>
    <p className="scan-progress">{progressFailed ? t("无法读取扫描进度，扫描仍在进行。") : progress ? t("已读取 {{size}} · {{files}} 个文件", { size: formatCapacity(progress.bytes), files: progress.files.toLocaleString(i18n.language) }) : t("正在准备…")}</p>
    {progress?.path && <p className="scan-folder"><Icon name="folder" /><span className="path">{progress.path}</span></p>}
    <p className="scan-hint">{t("进度按已读取容量与磁盘已用容量估算，不代表剩余时间。无法读取的位置会在结果中标明。")}</p>
    <button className="secondary-button" onClick={() => void stopScan()} disabled={stopping}><Icon name="close" />{t("停止扫描")}</button>
  </section>;
}

function Results({ summary, isMacOS, openDiskAccess, volumeName }: { summary: SpaceSummary; isMacOS: boolean; openDiskAccess: () => void; volumeName: string }) {
  const { t } = useTranslation();
  const { tab } = useSpace();
  const seconds = Math.max(1, Math.round((Date.parse(summary.finishedAt) - Date.parse(summary.startedAt)) / 1000));
  const tabs: [SpaceTab, string][] = [["folders", t("空间分布")], ["files", t("大文件")], ["suggestions", t("建议清理")]];
  return <>
    <div className="space-toolbar">
      <div className="segmented" role="tablist" aria-label={t("扫描结果")}>{tabs.map(([id, label]) => <button key={id} role="tab" aria-selected={tab === id} className={tab === id ? "selected" : ""} onClick={() => showTab(id)}>{label}</button>)}</div>
      <span className="space-scanned">{t("{{time}} 扫描 · 用时 {{seconds}} 秒", { time: timeLabel(summary.finishedAt), seconds })}</span>
      <button className="secondary-button" onClick={() => void scanSpace()}><Icon name="refresh" />{t("重新扫描")}</button>
    </div>
    {summary.skipped > 0 && <div className="space-access" role="status"><span>{t("{{count}} 个位置无法读取，结果可能偏小。", { count: summary.skipped })}</span>{isMacOS && <button className="cleanup-action" onClick={openDiskAccess}>{t("打开完全磁盘访问权限")}</button>}</div>}
    {tab === "folders" ? <Folders volumeName={volumeName} /> : tab === "files" ? <LargestFiles summary={summary} /> : <Suggestions summary={summary} />}
  </>;
}

function TrashButton({ entry }: { entry: SpaceEntry }) {
  const { t } = useTranslation();
  const { busy, summary } = useSpace();
  const requestConfirmation = useContext(SpaceConfirmation)!;
  if (!entry.trashable) return <span className="space-trash" aria-hidden="true" />;
  return <button className="space-trash" disabled={!!busy} aria-busy={busy === entry.path} aria-label={t("将「{{name}}」移入废纸篓", { name: entry.name })} title={t("移入废纸篓")} aria-haspopup="dialog" onClick={() => requestConfirmation({
    message: t("将「{{name}}」（{{size}}）移入废纸篓？目录内的全部内容都会一起移动。路径：{{path}}", { name: entry.name, size: formatCapacity(entry.bytes), path: entry.path }),
    confirm: () => void trashItem(entry, summary?.id),
  })}><Icon name="trash" /></button>;
}

function Folders({ volumeName }: { volumeName: string }) {
  const { t } = useTranslation();
  const { listing } = useSpace();
  if (!listing) return null;
  const crumbs = [...listing.ancestors, listing.entry];
  const label = (entry: SpaceEntry, index: number) => index === 0 ? volumeName : entry.name;
  const parent = listing.ancestors.at(-1);
  return <section className="space-card space-browser" aria-label={t("空间分布")}>
    <div className="space-browser-head">
      <button className="icon-button" disabled={!parent} onClick={() => parent && void openFolder(parent.path)} aria-label={t("返回上一级")} title={t("返回上一级")}><Icon name="back" /></button>
      <nav className="space-crumbs" aria-label={t("当前位置")}>{crumbs.map((entry, index) => index === crumbs.length - 1
        ? <strong key={entry.path} aria-current="page">{label(entry, index)}</strong>
        : <span key={entry.path}><button onClick={() => void openFolder(entry.path)}>{label(entry, index)}</button><Icon name="chevron" /></span>)}</nav>
      <span className="space-head-size"><strong>{formatCapacity(listing.entry.bytes)}</strong><small>{t("{{count}} 个文件", { count: listing.entry.files.toLocaleString() })}</small></span>
    </div>
    {listing.children.length ? <div className="space-rows">{listing.children.map(entry => <Row key={entry.other ? "other" : entry.path} entry={entry} whole={listing.entry.bytes} />)}</div>
      : <p className="space-empty">{t("此文件夹内没有超过 1 MB 的内容。")}</p>}
  </section>;
}

function Row({ entry, whole }: { entry: SpaceEntry; whole: number }) {
  const { t } = useTranslation();
  const ratio = share(entry.bytes, whole);
  const body = <>
    <span className={entry.directory ? "space-row-icon folder" : "space-row-icon"}><Icon name={entry.other ? "file" : entry.directory ? "folder" : "file"} /></span>
    <span className="space-row-name"><strong>{entry.other ? t("其他 {{count}} 个较小项目", { count: entry.files.toLocaleString() }) : entry.name}</strong>
      <small>{entry.other ? t("单个小于 1 MB，或不在前 200 项内") : entry.directory ? `${t("{{count}} 个文件", { count: entry.files.toLocaleString() })}${entry.partial ? ` · ${t("部分无法读取")}` : ""}` : t("文件")}</small></span>
    <span className="space-share" aria-hidden="true"><i style={{ width: `${ratio * 100}%` }} /></span>
    <span className="space-percent">{percent(ratio * 100)}</span>
    <span className="space-size">{formatCapacity(entry.bytes)}</span>
  </>;
  return <div className="space-row" data-other={entry.other || undefined}>
    {entry.directory ? <button className="space-row-main" onClick={() => void openFolder(entry.path)} title={entry.path}>{body}<span className="space-chevron"><Icon name="chevron" /></span></button>
      : <div className="space-row-main" title={entry.path || undefined}>{body}<span className="space-chevron" /></div>}
    <TrashButton entry={entry} />
  </div>;
}

function parentOf(entry: SpaceEntry, root: string) {
  const parent = entry.path.slice(0, entry.path.length - entry.name.length - 1);
  return parent.length < root.length ? root : parent;
}

function LargestFiles({ summary }: { summary: SpaceSummary }) {
  const { t } = useTranslation();
  const largest = summary.largest[0]?.bytes ?? 0;
  if (!summary.largest.length) return <section className="space-card"><p className="space-empty">{t("没有找到超过 1 MB 的文件。")}</p></section>;
  return <section className="space-card space-browser" aria-label={t("大文件")}>
    <div className="space-browser-head"><h2>{t("最大的 {{count}} 个文件", { count: summary.largest.length })}</h2><span className="space-head-size"><small>{t("不含废纸篓")}</small></span></div>
    <div className="space-rows">{summary.largest.map(entry => {
      const folder = parentOf(entry, summary.root);
      return <div className="space-row" key={entry.path}>
        <div className="space-row-main" title={entry.path}>
          <span className="space-row-icon"><Icon name="file" /></span>
          <span className="space-row-name"><strong>{entry.name}</strong><small className="path">{folder}</small></span>
          <span className="space-share" aria-hidden="true"><i style={{ width: `${share(entry.bytes, largest) * 100}%` }} /></span>
          <span className="space-size">{formatCapacity(entry.bytes)}</span>
          <button className="space-reveal" onClick={() => void openFolder(folder)} aria-label={t("在空间分布中查看「{{name}}」", { name: entry.name })} title={t("在空间分布中查看")}><Icon name="reveal" /></button>
        </div>
        <TrashButton entry={entry} />
      </div>;
    })}</div>
  </section>;
}

function Suggestions({ summary }: { summary: SpaceSummary }) {
  const { t } = useTranslation();
  const { busy } = useSpace();
  const requestConfirmation = useContext(SpaceConfirmation)!;
  const total = summary.suggestions.reduce((sum, item) => sum + item.bytes, 0);
  return <section className="space-card space-browser" aria-label={t("建议清理")}>
    <div className="space-browser-head"><h2>{t("常见可清理位置")}</h2><span className="space-head-size"><strong>{formatCapacity(total)}</strong><small>{t("合计")}</small></span></div>
    <div className="space-rows">{summary.suggestions.map(item => <div className="space-row" key={item.path}>
      <div className="space-row-main">
        <span className="space-suggestion-icon" data-kind={item.trash ? "trash" : "folder"}><Icon name={item.trash ? "trash" : item.name === "下载文件" ? "hide" : "cleanup"} /></span>
        <span className="space-row-name"><strong>{t(item.name)}</strong><small className="path">{item.path}</small></span>
        <span className="space-size">{!item.found ? t("未找到") : item.bytes < 1_000_000 ? "< 1 MB" : formatCapacity(item.bytes)}</span>
      </div>
      <div className="space-suggestion-actions">
        {item.found && item.bytes >= 1_000_000 && <button className="cleanup-action" onClick={() => void openFolder(item.path)}>{t("查看")}</button>}
        {item.trash && <button className="space-danger" disabled={!!busy || !item.found} aria-busy={busy === "trash" || busy === "trash-review"} aria-haspopup="dialog"
          onClick={async event => {
            const trigger = event.currentTarget;
            const review = await prepareTrash();
            if (!review) return;
            if (!trigger.isConnected) {
              void discardTrashReview(review.id);
              return;
            }
            requestConfirmation({
              message: t("永久删除废纸篓中已复核的 {{count}} 项，预计释放 {{size}}？此操作不可恢复。位置 {{path}}", { count: review.count, size: formatBytes(review.bytes), path: review.path }),
              details: <TrashManifest review={review} />,
              cancel: () => void discardTrashReview(review.id),
              confirm: () => void emptyTrash(review), permanent: true,
            });
          }}>{busy === "trash-review" ? t("正在复核废纸篓…") : t("清空废纸篓")}</button>}
      </div>
    </div>)}</div>
    <p className="space-footnote">{t("缓存和日志可能仍被 App 使用，删除后 App 会按需重建。打开文件夹可逐项移入废纸篓。")}</p>
  </section>;
}
