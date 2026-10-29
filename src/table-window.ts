// Matches the fixed activity-table row and header geometry in style.css.
export const activityRowHeight = 26;
const activityHeaderHeight = 32;
export interface TableWindow { start: number; count: number }

export function tableWindow(scrollTop: number, height: number): TableWindow {
  const visible = Math.max(1, Math.ceil(height / activityRowHeight));
  // One viewport on either side keeps wheel scrolling and Tab focus continuous.
  return { start: Math.max(0, Math.floor((scrollTop - activityHeaderHeight) / activityRowHeight) - visible), count: visible * 3 };
}

export function windowRows<T>(rows: T[], window?: TableWindow) {
  const start = window ? Math.min(window.start, Math.max(0, rows.length - window.count)) : 0;
  const end = window ? Math.min(rows.length, start + window.count) : rows.length;
  return { rows: rows.slice(start, end), start, before: start * activityRowHeight, after: (rows.length - end) * activityRowHeight };
}
