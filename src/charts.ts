import { t } from "./i18n";
import type { Trend } from "./mygo";
import { escapeHTML as e } from "./format";

export type TrendKey = "cpu" | "memory" | "gpu" | "diskRead" | "diskWrite" | "download" | "upload" | "battery" | "power";

export function bucketHistory(history: Trend[], keys: TrendKey[], end: number, seconds: number, count: number): (number[] | null)[] {
  const start = end - seconds * 1000;
  const buckets: { sums: number[]; count: number }[] = Array.from({ length: count }, () => ({ sums: keys.map(() => 0), count: 0 }));
  for (const point of history) {
    const at = new Date(point.at).getTime();
    if (at < start || at > end) continue;
    const values = keys.map(key => point[key]);
    if (values.some(value => value == null)) continue;
    const bin = Math.min(count - 1, Math.floor((at - start) / (seconds * 1000) * count));
    for (let i = 0; i < keys.length; i++) buckets[bin]!.sums[i] = buckets[bin]!.sums[i]! + values[i]!;
    buckets[bin]!.count++;
  }
  return buckets.map(bucket => bucket.count ? bucket.sums.map(sum => sum / bucket.count) : null);
}

export function dots(history: Trend[], keys: TrendKey[], end: number, seconds: number, label: string, max = 100, geometry = { columns: 36, rows: 5, radius: 2.75 }): string {
  const { columns, rows, radius } = geometry;
  const data = bucketHistory(history, keys, end, seconds, columns);
  const ceiling = max || Math.max(1, ...data.map(values => values?.reduce((a, b) => a + b, 0) || 0));
  const circles: string[] = [];
  for (let x = 0; x < columns; x++) {
    const values = data[x];
    const total = values?.reduce((a, b) => a + b, 0) || 0;
    for (let y = 0; y < rows; y++) {
      const level = (rows - y) / rows;
      const filled = values != null && total > 0 && level <= Math.max(1 / rows, total / ceiling);
      const secondary = filled && values!.length > 1 && level <= values![1]! / ceiling;
      circles.push(`<circle cx="${((x + .5) / columns * 100).toFixed(3)}%" cy="${(y + .5) / rows * 100}%" r="${radius}" class="${filled ? secondary ? "dot-secondary" : "dot-active" : values ? "dot-idle" : "dot-empty"}"/>`);
    }
  }
  return `<div class="dot-chart"><svg role="img" aria-label="${e(label)}${t("。淡色空白列表示该时段尚无样本。")}">${circles.join("")}</svg></div>`;
}

export function lineChart(history: Trend[], key: TrendKey, end: number, seconds: number, label: string, max = 100, secondary?: TrendKey, interval = 2): string {
  const keys = secondary ? [key, secondary] : [key];
  const start = end - seconds * 1000;
  const data = history.filter(p => new Date(p.at).getTime() >= start && new Date(p.at).getTime() <= end);
  const ceiling = max || Math.max(1, ...data.flatMap(p => keys.map(k => p[k] || 0)));
  const series = keys.map((key, i) => {
    let path = "", drawing = false, previous = 0;
    for (const point of data) {
      const value = point[key], at = new Date(point.at).getTime();
      if (value == null) { drawing = false; continue; }
      if (at - previous > interval * 3000) drawing = false;
      path += `${drawing ? "L" : "M"}${((at - start) / (seconds * 1000) * 900).toFixed(1)},${(160 - Math.min(1, value / ceiling) * 148).toFixed(1)} `;
      drawing = true;
      previous = at;
    }
    return `<path d="${path}" class="${i ? "series-secondary" : "series-primary"}"/>`;
  }).join("");
  const hasData = data.filter(p => p[key] != null).length > 1;
  return `<div class="line-chart"><svg viewBox="0 0 900 170" preserveAspectRatio="none" role="img" aria-label="${e(label)}"><path class="grid-line" d="M0 12H900M0 86H900M0 160H900"/>${series}</svg>${!hasData ? `<span class="chart-empty">${t("等待两个有效样本")}</span>` : ""}</div><div class="chart-axis"><span>${seconds >= 3600 ? t("1 小时") : seconds / 60 + t(" 分钟")}${t("前")}</span><span>${t("现在")}</span></div>`;
}
