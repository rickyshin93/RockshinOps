import type { Trend } from "./mygo";

// Samples are immutable; all charts share their parsed timestamps.
const timestamps = new WeakMap<Trend, number>();
export function trendTime(point: Trend): number {
  let time = timestamps.get(point);
  if (time === undefined) { time = Date.parse(point.at); timestamps.set(point, time); }
  return time;
}

export function mergeHistory(current: Trend[], incoming: Trend[], end: number, seconds: number): Trend[] {
  const cutoff = end - seconds * 1000;
  const retained = current.filter(point => trendTime(point) >= cutoff && trendTime(point) <= end);
  const last = retained.length ? trendTime(retained[retained.length - 1]!) : -Infinity;
  return retained.concat(incoming.filter(point => trendTime(point) > last && trendTime(point) >= cutoff && trendTime(point) <= end));
}
