import type { ReactNode } from "react";

export function Group({ id, title, note, children }: { id: string; title: string; note?: string; children: ReactNode }) {
  return <section className="settings-group" aria-labelledby={id}><h2 id={id}>{title}</h2><div className="settings-card">{children}</div>{note && <p className="settings-note">{note}</p>}</section>;
}

export function Row({ id, label, description, lead, children }: { id: string; label: string; description?: string; lead?: ReactNode; children: ReactNode }) {
  return <div className={lead ? "settings-row with-lead" : "settings-row"}>{lead}<div className="setting-copy"><span id={id} className="setting-label">{label}</span>{description && <p>{description}</p>}</div><div className="setting-control">{children}</div></div>;
}

export function Switch({ on, labelledBy, metric, action, disabled = false }: { on: boolean; labelledBy: string; metric?: string; action?: string; disabled?: boolean }) {
  return <button type="button" className="switch" disabled={disabled} role="switch" aria-checked={on} aria-labelledby={labelledBy} data-metric={metric} data-action={action}><span /></button>;
}
