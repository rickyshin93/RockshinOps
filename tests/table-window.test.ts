import {describe, expect, test} from 'bun:test';
import {activityRowHeight, tableWindow, windowRows} from '../src/table-window';

describe('activity table viewport', () => {
  const rows = Array.from({length:10000}, (_, i) => i);
  test('keeps every visible row covered without rendering the full list', () => {
    for (const height of [260, 567, 1000]) for (const scroll of [0, 31, 100, 12000, 250000]) {
      const view = windowRows(rows, tableWindow(scroll, height));
      const first = Math.max(0, Math.floor((scroll - 32) / activityRowHeight));
      const last = Math.min(rows.length - 1, Math.ceil((scroll + height - 32) / activityRowHeight));
      expect(view.rows).toContain(first);
      expect(view.rows).toContain(last);
      expect(view.rows.length).toBeLessThanOrEqual(Math.ceil(height / activityRowHeight) * 3);
      expect(view.before + view.rows.length * activityRowHeight + view.after).toBe(rows.length * activityRowHeight);
    }
  });
  test('a search shrinking the list while scrolled still shows its matches', () => {
    const view = windowRows([10, 20, 30], tableWindow(20000, 500));
    expect(view.rows).toEqual([10, 20, 30]);
    expect(view.before).toBe(0);
    expect(view.after).toBe(0);
    expect(windowRows([], tableWindow(20000, 500)).rows).toEqual([]);
  });
  test('clamps the bottom and preserves the source data', () => {
    const view = windowRows(rows, tableWindow(999999, 500));
    expect(view.rows.at(-1)).toBe(9999);
    expect(view.after).toBe(0);
    expect(rows[0]).toBe(0);
    expect(windowRows([1, 2]).rows).toEqual([1, 2]);
  });
});
