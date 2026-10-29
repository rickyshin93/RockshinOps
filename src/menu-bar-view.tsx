import { memo, useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode, type KeyboardEvent, type ButtonHTMLAttributes } from "react";
import { useTranslation } from "react-i18next";
import { System, type MenuBarConfiguration, type Settings as MenuBarSettings, type State as MenuBarState, type Item as MenuBarItem, type Appearance } from "./mygo";
import { icon } from "./icons";
import { moveMenuBarItem, reconcileMenuBarState } from "./menu-bar-layout";
import { MenuBarIcon } from "./menu-bar-icon";

const sections = [["visible", "可见"], ["hidden", "隐藏"], ["alwaysHidden", "始终隐藏"]] as const;
const tabs = [["layout", "布局"], ["general", "通用"], ["appearance", "外观"], ["hotkeys", "快捷键"], ["advanced", "高级"]] as const;
type Tab = typeof tabs[number][0];
function ActionButton({ blocked, disabled, ...props }: ButtonHTMLAttributes<HTMLButtonElement> & { blocked: boolean }) {
  return <button {...props} disabled={disabled || blocked} data-pending={blocked && !disabled || undefined} />;
}
function Row({ label, description, children }: { label: string; description?: string; children: ReactNode }) {
  return <div className="settings-row"><div className="setting-copy"><span className="setting-label">{label}</span>{description && <p>{description}</p>}</div><div className="setting-control">{children}</div></div>;
}
function Toggle({ label, on, disabled, change }: { label: string; on: boolean; disabled: boolean; change: () => void }) {
  return <button type="button" className="switch" role="switch" aria-label={label} aria-checked={on} data-pending={disabled || undefined} disabled={disabled} onClick={change}><span /></button>;
}
function Options({ label, value, choices, disabled, change }: { label: string; value: string; choices: readonly (readonly [string, string])[]; disabled: boolean; change: (v: string) => void }) {
  return <div className="segmented" role="radiogroup" aria-label={label}>{choices.map(([id, title]) => <label key={id}><input type="radio" checked={value === id} name={label} disabled={disabled} onChange={() => change(id)} /><span>{title}</span></label>)}</div>;
}
function Hotkey({ label, value, disabled, change }: { label: string; value: string; disabled: boolean; change: (v: string) => void }) {
  const { t } = useTranslation();
  const [recording, setRecording] = useState(false);
  function record(e: KeyboardEvent<HTMLButtonElement>) {
    if (!recording) return;
    e.preventDefault(); e.stopPropagation();
    if (e.key === "Escape") { setRecording(false); return; }
    if (["Meta", "Control", "Alt", "Shift"].includes(e.key)) return;
    if (!e.metaKey && !e.ctrlKey && !e.altKey) return;
    const key = e.code.startsWith("Key") ? e.code.slice(3) : e.code.startsWith("Digit") ? e.code.slice(5) : e.key === " " ? "Space" : e.key;
    change([e.metaKey && "Cmd", e.ctrlKey && "Ctrl", e.altKey && "Alt", e.shiftKey && "Shift", key].filter(Boolean).join("+")); setRecording(false);
  }
  return <div className="menu-hotkey"><button type="button" className="secondary-button" data-pending={disabled || undefined} disabled={disabled} aria-label={`${label}: ${value || t("未设置")}`} onClick={e => { e.currentTarget.focus(); setRecording(true); }} onKeyDown={record} onBlur={() => setRecording(false)}>{recording ? t("按下组合键，Esc 取消") : value || t("录制快捷键")}</button>{value && <button type="button" className="text-button" data-pending={disabled || undefined} disabled={disabled} aria-label={`${t("清除")}: ${label}`} onClick={() => change("")}>{t("清除")}</button>}</div>;
}
export const MenuBarView = memo(function MenuBarView({ isMacOS, refreshSeconds }: { isMacOS: boolean; refreshSeconds: number }) {
  const { t } = useTranslation();
  const [config, setConfig] = useState<MenuBarConfiguration | null>(null);
  const [tab, setTab] = useState<Tab>("layout");
  const [busy, setBusy] = useState(false);
  const operation = useRef(false);
  const reading = useRef(false);
  const revision = useRef(0);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<MenuBarItem | null>(null);
  const [drag, setDrag] = useState<MenuBarItem | null>(null);
  const drop = useRef<{ section: string; before: number; target: HTMLElement | null } | null>(null);
  const dragging = useRef(false);
  const [moving, setMoving] = useState<number | null>(null);
  const layout = useRef<HTMLDivElement>(null);
  const positions = useRef<Map<number, { x: number; y: number; section: string }> | null>(null);
  const [theme, setTheme] = useState<"light" | "dark">("light");
  const [message, setMessage] = useState("");
  useEffect(() => {
    if (tab !== "layout") return;
    const cancelSelection = (event: globalThis.KeyboardEvent) => { if (event.key === "Escape") setSelected(null); };
    window.addEventListener("keydown", cancelSelection);
    return () => window.removeEventListener("keydown", cancelSelection);
  }, [tab]);
  useEffect(() => {
    if (!isMacOS) return;
    let active = true;
    const version = revision.current;
    System.menuBarConfiguration().then(async value => { if (active) { setConfig(value); setError(value.state.error); if (value.state.screenRecording) { const snapshot = await System.menuBarAction("refresh"); if(active && version === revision.current) updateState(snapshot); } } }).catch(reason => { if (active && version === revision.current) setError(String(reason)); });
    return () => { active = false; };
  }, [isMacOS]);
  useEffect(() => {
    if ((!config?.settings.enabled && !config?.state.capturing) || busy || drag || moving !== null) return;
    let active = true;
    let timer: number;
    async function poll() {
      if (!active) return;
      const version = revision.current;
      if (!document.hidden && !reading.current && !operation.current && !dragging.current) {
        reading.current = true;
        try {
          const snapshot = await System.menuBarAction("state");
          if (active && version === revision.current && !operation.current && !dragging.current) updateState(snapshot);
        } catch (reason) { if (active && version === revision.current) setError(String(reason)); }
        finally { reading.current = false; }
      }
      if (active) timer = window.setTimeout(poll, refreshSeconds * 1000);
    }
    timer = window.setTimeout(poll, refreshSeconds * 1000);
    return () => { active = false; window.clearTimeout(timer); };
  }, [config?.settings.enabled, config?.state.capturing, busy, drag, moving, refreshSeconds]);
  function updateState(snapshot: MenuBarState) {
    setConfig(old => {
      if (!old) return old;
      const next = reconcileMenuBarState(old.state, snapshot);
      return next === old.state ? old : { ...old, state: next };
    });
  }
  function beginOperation() {
    if (operation.current || dragging.current) return false;
    operation.current = true;
    revision.current++;
    return true;
  }
  const settings = config?.settings;
  const state = config?.state;
  const itemGroups = useMemo(() => {
    const search = query.toLocaleLowerCase();
    return sections.map(([id, title]) => ({ id, title, items: state?.items.filter(item => item.section === id && `${item.name} ${item.owner}`.toLocaleLowerCase().includes(search)) || [] }));
  }, [state?.items, query]);
  useLayoutEffect(() => {
    const previous = positions.current;
    positions.current = null;
    if (!previous || window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
    // Read all positions before starting animations; only the requested move animates.
    const changes = Array.from(layout.current?.querySelectorAll<HTMLButtonElement>("[data-menu-id]") || []).map(button => ({
      button, x: button.offsetLeft, y: button.offsetTop, old: previous.get(Number(button.dataset.menuId)),
    }));
    for (const { button, x, y, old } of changes) {
      if (old && old.section === button.dataset.section && (old.x !== x || old.y !== y)) {
        button.animate([{ transform: `translate(${old.x - x}px, ${old.y - y}px)` }, { transform: "translate(0, 0)" }], { duration: 160, easing: "cubic-bezier(.2,.8,.2,1)" });
      }
    }
  }, [state?.items]);
  async function run(action: () => Promise<MenuBarState>) {
    if (!beginOperation()) return;
    setBusy(true); setError(""); setMessage("");
    try { updateState(await action()); }
    catch (reason) { setError(String(reason)); }
    finally { operation.current = false; setBusy(false); }
  }
  async function reload() {
    if (!beginOperation()) return;
    setBusy(true); setError("");
    try { const next = await System.menuBarConfiguration(); setConfig(next); setError(next.state.error); if(next.state.screenRecording) {const snapshot = await System.menuBarAction("refresh");updateState(snapshot);} }
    catch (reason) { setError(String(reason)); }
    finally { operation.current = false; setBusy(false); }
  }
  async function save(patch: Partial<MenuBarSettings>) {
    if (!settings || !beginOperation()) return;
    setBusy(true); setError(""); setMessage("");
    try { setConfig(await System.saveMenuBarSettings({ ...settings, ...patch })); setMessage(t("已保存")); }
    catch (reason) { setError(`${t("保存失败，设置已恢复：")}${String(reason)}`); }
    finally { operation.current = false; setBusy(false); }
  }
  function toggle(key: keyof MenuBarSettings, label: string, description?: string) {
    return <Row label={t(label)} description={description && t(description)}><Toggle label={t(label)} on={Boolean(settings?.[key])} disabled={busy} change={() => void save({ [key]: !settings?.[key] })} /></Row>;
  }
  async function move(item: MenuBarItem, section: string, before = 0) {
    if (operation.current || item.id === before) return;
    finishDrag(); beginOperation(); setMoving(item.id); setError("");
    const previous = state!;
    positions.current = new Map(Array.from(layout.current?.querySelectorAll<HTMLButtonElement>("[data-menu-id]") || []).map(button => [Number(button.dataset.menuId), { x: button.offsetLeft, y: button.offsetTop, section: button.dataset.section! }]));
    const preview = moveMenuBarItem(previous.items, item.id, section, before);
    setConfig(old => old ? { ...old, state: { ...old.state, items: preview } } : old);
    setSelected(old => old ? preview.find(i => i.id === old.id) || null : null);
    try {
      const next = await System.moveMenuBarItem(item.id, item.pid, section, before);
      setConfig(old => old ? { ...old, state: next } : old);
      setSelected(old => old ? next.items.find(i => i.id === old.id) || null : null);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
      setConfig(old => old ? { ...old, state: previous } : old);
      setSelected(old => old ? previous.items.find(i => i.id === old.id) || null : null);
      try {
        const actual = await System.menuBarAction("refresh");
        setConfig(old => old ? { ...old, state: actual } : old);
        setSelected(old => old ? actual.items.find(i => i.id === old.id) || null : null);
      } catch (refreshError) { setError(`${reason instanceof Error ? reason.message : String(reason)} · ${refreshError instanceof Error ? refreshError.message : String(refreshError)}`); }
    } finally { operation.current = false; setMoving(null); }
  }
  function dropTarget(section: string, before: number, target: HTMLElement | null = null) {
    if (drop.current?.target !== target) {
      drop.current?.target?.removeAttribute("data-drop-target");
      target?.setAttribute("data-drop-target", "true");
    }
    drop.current = { section, before, target };
  }
  function finishDrag() {
    dragging.current = false;
    drop.current?.target?.removeAttribute("data-drop-target");
    drop.current = null;
    setDrag(null);
  }
  function appearance(a: Appearance) {
    const update = (patch: Partial<Appearance>) => void save({ [settings?.dynamic ? theme : "light"]: { ...a, ...patch } });
    return <div className="settings-card">
      <Row label={t("色调")}><Options label={t("色调")} value={a.tint} disabled={busy} choices={[["none", t("无")], ["solid", t("纯色")], ["gradient", t("渐变")]]} change={tint => update({ tint })} /></Row>
      {a.tint !== "none" && <Row label={t("颜色")}><input className="menu-color" type="color" aria-label={t("颜色")} value={a.color} disabled={busy} onChange={e => update({ color: e.target.value })} /></Row>}
      <Row label={t("阴影")}><Toggle label={t("阴影")} on={a.shadow} disabled={busy} change={() => update({ shadow: !a.shadow })} /></Row>
      <Row label={t("边框")}><Toggle label={t("边框")} on={a.border} disabled={busy} change={() => update({ border: !a.border })} /></Row>
      <Row label={t("菜单栏形状")}><Options label={t("菜单栏形状")} value={a.shape} disabled={busy} choices={[["none", t("无")], ["rounded", t("圆角")], ["split", t("分段")]]} change={shape => update({ shape })} /></Row>
    </div>;
  }
  if (!isMacOS) return <div className="empty-state"><h2>{t("菜单栏管理仅支持 macOS")}</h2><p>{t("系统监控与托盘读数仍可正常使用。")}</p></div>;
  if (!config) return <div className="empty-state"><h2>{t("读取菜单栏")}</h2><p role={error ? "alert" : "status"}>{error || t("正在读取本机偏好…")}</p>{error && <ActionButton blocked={busy || moving !== null} className="secondary-button" onClick={() => void reload()}>{t("重试")}</ActionButton>}</div>;
  const editable = Boolean(settings?.enabled && state?.accessibility && !state.iceRunning);
  return <div className="menu-manager" aria-busy={busy || moving !== null}>
    <div className="settings-card menu-enable">{toggle("enabled", "启用菜单栏管理", "将不常用图标收进隐藏分区，点击菜单栏箭头随时展开。")}</div>
    {state?.iceRunning && <div className="notice">{t("请先退出 Ice，避免两个菜单栏管理器同时运行。")}</div>}
    {error && <div className="notice error" role="alert">{error}</div>}
    <div className="menu-toolbar">
      <div className="segmented menu-tabs" role="tablist" aria-label={t("菜单栏管理")}>{tabs.map(([id, label]) => <button key={id} id={`menu-tab-${id}`} tabIndex={tab === id ? 0 : -1} onKeyDown={e => { if (["ArrowLeft", "ArrowRight", "Home", "End"].includes(e.key)) { e.preventDefault(); const index=tabs.findIndex(([key])=>key===id); const next=e.key==="Home"?0:e.key==="End"?tabs.length-1:(index+(e.key==="ArrowRight"?1:tabs.length-1))%tabs.length; setTab(tabs[next]![0]); document.getElementById(`menu-tab-${tabs[next]![0]}`)?.focus(); } }} role="tab" aria-selected={tab === id} aria-controls="menu-tab-panel" className={tab === id ? "selected" : ""} onClick={() => setTab(id)}>{t(label)}</button>)}</div>
      <ActionButton blocked={busy || moving !== null} type="button" className="secondary-button" onClick={() => void reload()}><span dangerouslySetInnerHTML={{ __html: icon("refresh") }} />{t("刷新")}</ActionButton>
    </div>
    <div id="menu-tab-panel" role="tabpanel" aria-labelledby={`menu-tab-${tab}`}>
      {tab === "layout" && <div className="menu-layout" ref={layout} onPointerDown={e => { if (!(e.target as Element).closest("button, input, label")) setSelected(null); }}>
        <div className="menu-layout-actions"><label className="menu-search"><span dangerouslySetInnerHTML={{ __html: icon("search") }} /><input aria-label={t("搜索菜单栏图标")} placeholder={t("搜索菜单栏图标")} value={query} onChange={e => setQuery(e.target.value)} /></label>
          <ActionButton blocked={busy || moving !== null} className="secondary-button" disabled={!settings?.enabled} onClick={() => void run(() => System.menuBarAction("all"))}>{t("显示全部分区")}</ActionButton>
          <ActionButton blocked={busy || moving !== null} className="secondary-button" disabled={!settings?.enabled} onClick={() => void run(() => System.menuBarAction(state?.expanded ? "collapse" : "expand"))}>{state?.expanded ? t("收起") : t("展开")}</ActionButton>
          <ActionButton blocked={busy || moving !== null} className="secondary-button" disabled={!settings?.enabled || !state?.screenRecording} onClick={() => void run(() => System.menuBarAction("panel"))}>{t("打开展开栏")}</ActionButton>
          <ActionButton blocked={busy || moving !== null} className="secondary-button" aria-label={t("搜索菜单栏图标")} title={t("搜索菜单栏图标")} disabled={!settings?.enabled || !state?.screenRecording} onClick={() => void run(() => System.menuBarAction("search"))}><span dangerouslySetInnerHTML={{ __html: icon("search") }} /></ActionButton>
        </div>
        {!state?.accessibility && <p className="menu-help">{t("需要辅助功能权限以操作菜单栏图标。")}</p>}
        {!state?.screenRecording && <p className="menu-help">{t("授权屏幕录制后可显示真实图标；当前显示应用名称。")}</p>}
        {itemGroups.filter(({ id }) => id !== "alwaysHidden" || settings?.alwaysHidden).map(({ id, title, items }) => {
          return <section className="menu-section" key={id} aria-labelledby={`section-${id}`}><div className="menu-section-heading"><h2 id={`section-${id}`}>{t(title)}<span>{items.length}</span></h2>{selected && <ActionButton blocked={busy || moving !== null} className="secondary-button" disabled={!editable || !selected.movable || selected.section === id} onClick={() => move(selected, id)}>{t("移到此分区")}</ActionButton>}</div>
            <div className="menu-strip" onDragOver={e => { if (editable) { e.preventDefault(); dropTarget(id, 0); } }} onDrop={e => { e.preventDefault(); if (drag && editable) void move(drag, id); }}>
              {items.length ? items.map(item => <button key={item.id} type="button" data-menu-id={item.id} data-section={id} className={`menu-item${selected?.id === item.id ? " selected" : ""}${drag?.id === item.id ? " dragging" : ""}${moving === item.id ? " moving" : ""}`} draggable={editable && item.movable} aria-pressed={selected?.id === item.id} aria-label={item.name} title={item.owner === item.name ? item.name : `${item.owner} · ${item.name}`} onPointerDown={e => { if (e.button === 0) { e.currentTarget.focus({ preventScroll: true }); setSelected(old => old?.id === item.id ? null : item); } }} onClick={e => { if (e.detail === 0) setSelected(old => old?.id === item.id ? null : item); }} onDragStart={e => { if (operation.current) { e.preventDefault(); return; } revision.current++; dragging.current = true; setSelected(item); layout.current?.querySelectorAll("[data-menu-id]").forEach(button => button.getAnimations().forEach(animation => animation.cancel())); e.dataTransfer.effectAllowed = "move"; e.dataTransfer.setData("text/plain", String(item.id)); setDrag(item); }} onDragEnd={finishDrag} onDragOver={e => { if (!editable) return; e.preventDefault(); e.stopPropagation(); const rect = e.currentTarget.getBoundingClientRect(); const next = items[items.findIndex(i => i.id === item.id) + 1]; dropTarget(id, e.clientX < rect.x + rect.width / 2 ? item.id : next?.id || 0, e.currentTarget); }} onDrop={e => { e.preventDefault(); e.stopPropagation(); if (drag && editable) void move(drag, id, drop.current?.section === id ? drop.current.before : item.id); }}><MenuBarIcon image={item.image} name={item.name} /></button>) : <span className="menu-empty">{query ? t("没有匹配图标") : t("此分区没有图标")}</span>}
            </div>
          </section>;
        })}
        <div className="menu-selection">{selected ? <><span className="menu-selection-name">{selected.name}{selected.owner !== selected.name && <span>{selected.owner}</span>}</span><ActionButton blocked={busy || moving !== null} className="secondary-button" disabled={!editable} onClick={() => void run(() => System.activateMenuBarItem(selected.id))}>{t("打开图标菜单")}</ActionButton>{(["previous", "next"] as const).map(direction => { const same = state!.items.filter(i => i.section === selected.section); const index = same.findIndex(i => i.id === selected.id); const target = direction === "previous" ? same[index - 1] : same[index + 2]; return <ActionButton blocked={busy || moving !== null} key={direction} className="secondary-button" disabled={!editable || !selected.movable || (direction === "previous" ? index <= 0 : index >= same.length - 1)} onClick={() => move(selected, selected.section, target?.id || 0)}>{direction === "previous" ? t("向左移动") : t("向右移动")}</ActionButton>; })}<button className="menu-clear" onClick={() => setSelected(null)}>{t("取消选择")}</button></> : <span className="menu-selection-hint">{t("点击图标选择，拖动调整位置")}</span>}</div>
      </div>}
      {tab === "general" && <div className="settings-card">
        {toggle("usePanel", "使用独立展开栏", "在菜单栏下方显示隐藏图标，适合带刘海的显示器。")}
        {toggle("showOnHover", "悬停时展开", "将指针停在菜单栏箭头上自动展开隐藏图标。")}
        <Row label={t("单个图标停留时长")} description={t("临时显示的图标至少保留此时长（5–60 秒）；菜单打开期间不会收回。") }><input key={settings?.temporaryRevealSeconds} aria-label={t("单个图标停留时长")} className="menu-number" type="number" min="5" max="60" step="1" required defaultValue={settings?.temporaryRevealSeconds} disabled={busy} onBlur={e => { if (!e.target.reportValidity()) return; if (Number(e.target.value) !== settings?.temporaryRevealSeconds) void save({ temporaryRevealSeconds: Number(e.target.value) }); }} /><span>{t("秒")}</span></Row>
        <Row label={t("自动收起延迟")} description={t("设为 0 表示不自动收起；菜单打开或正在拖动时不会收起。")}><input key={settings?.rehideDelay} aria-label={t("自动收起延迟")} className="menu-number" type="number" min="0" max="300" defaultValue={settings?.rehideDelay} disabled={busy} onBlur={e => { if (e.target.checkValidity() && Number(e.target.value) !== settings?.rehideDelay) void save({ rehideDelay: Number(e.target.value) }); }} /><span>{t("秒")}</span></Row>
      </div>}
      {tab === "appearance" && <div className="menu-appearance"><div className="settings-card">{toggle("dynamic", "跟随系统外观", "为系统浅色和深色菜单栏分别配置外观。")}</div>{settings?.dynamic && <Options label={t("编辑外观")} value={theme} choices={[["light", t("浅色")], ["dark", t("深色")]]} disabled={busy} change={value => setTheme(value as "light" | "dark")} />}{settings && appearance(settings[settings.dynamic ? theme : "light"])}<p className="settings-note">{t("外观应用到系统菜单栏，不影响图标点击；全屏与自动隐藏行为以 macOS 为准。")}</p></div>}
      {tab === "hotkeys" && <div className="settings-card">{([["toggleHotkey", "展开或收起隐藏图标"], ["searchHotkey", "搜索菜单栏图标"], ["panelHotkey", "打开展开栏"], ["menusHotkey", "切换应用菜单"]] as const).map(([key, label]) => <Row key={key} label={t(label)}><Hotkey label={t(label)} value={settings?.[key] || ""} disabled={busy} change={value => void save({ [key]: value })} /></Row>)}</div>}
      {tab === "advanced" && <div className="menu-advanced"><div className="settings-card">
        {toggle("alwaysHidden", "启用始终隐藏分区", "普通展开不显示此分区；使用显示全部分区访问。")}
        {toggle("showAllOnDrag", "Command 拖动时显示全部分区")}
        <Row label={t("分隔符样式")}><Options label={t("分隔符样式")} value={settings?.divider || "line"} disabled={busy} choices={[["none", t("无")], ["line", t("线条")], ["dot", t("圆点")]]} change={divider => void save({ divider })} /></Row>
        {toggle("hideAppMenus", "展开时隐藏应用菜单", "展开时暂时切换到 RockshinOps 的空菜单，收起后恢复原应用。")}
        {toggle("secondaryMenu", "空白区域右键菜单", "在系统菜单栏空白处右键打开分区操作。")}
        <Row label={t("悬停延迟")}><input key={settings?.hoverDelay} aria-label={t("悬停延迟")} className="menu-number" type="number" min="0" max="5" step="0.1" defaultValue={settings?.hoverDelay} disabled={busy} onBlur={e => { if (e.target.checkValidity() && Number(e.target.value) !== settings?.hoverDelay) void save({ hoverDelay: Number(e.target.value) }); }} /><span>{t("秒")}</span></Row>
      </div></div>}
    </div>
    <section className="settings-group menu-permissions" aria-labelledby="menu-permissions"><h2 id="menu-permissions">{t("权限")}</h2><div className="settings-card">{([["accessibility", "辅助功能", "允许移动和打开其他应用的菜单栏图标。"], ["screenRecording", "屏幕录制", "仅捕获菜单栏图标窗口，用于布局预览与展开栏。"]] as const).map(([key, label, description]) => <Row key={key} label={t(label)} description={t(description)}>{state?.[key] ? <span className="menu-granted">{t("已授权")}</span> : <ActionButton blocked={busy || moving !== null} className="secondary-button" onClick={() => void run(() => System.menuBarPermission(key))}>{t("授权")}</ActionButton>}</Row>)}</div>{(!state?.accessibility || !state?.screenRecording) && <p className="settings-note">{t("授权后点击刷新；macOS 可能要求退出并重新打开应用。")}</p>}</section>
    <span className="menu-save-status" role="status" aria-live="polite">{busy || moving !== null ? t("正在应用…") : message}</span>
  </div>;
});
