import { describe, expect, test } from "bun:test";
import { bucketHistory, dots, lineChart } from "../src/charts";
import type { Trend } from "../src/mygo";
const end = Date.parse("2026-10-26T12:00:00Z");
function point(secondsAgo: number, cpu: number | null): Trend {
  return { at: new Date(end - secondsAgo * 1000).toISOString(), cpu, memory: null, gpu: null, diskRead: null, diskWrite: null, download: null, upload: null, battery: null, power: null };
}
describe("real history rendering", () => {
  test("compact process dots leave unrecorded time empty and use only measured samples", () => {
    const geometry = { columns: 42, rows: 6, radius: 1.7 };
    const empty = dots([], ["cpu"], end, 60, "CPU", 100, geometry);
    expect(empty.match(/class="dot-empty"/g)?.length).toBe(252);
    expect(empty).not.toContain('class="dot-active"');
    const sampled = dots([point(0, 50)], ["cpu"], end, 60, "CPU", 100, geometry);
    expect(sampled.match(/class="dot-active"/g)?.length).toBe(3);
    expect(sampled.match(/class="dot-empty"/g)?.length).toBe(246);
  });
  test("does not fill unrecorded buckets or coerce unavailable readings into zero", () => {
    expect(bucketHistory([point(9, 0), point(8, 20), point(5, null), point(0, 80)], ["cpu"], end, 10, 5)).toEqual([[0], [20], null, null, [80]]);
  });
  test("excludes points outside the selected time range", () => {
    expect(bucketHistory([point(11, 90), point(-1, 90), point(0, 10)], ["cpu"], end, 10, 2)).toEqual([null, [10]]);
  });
  test("breaks paths across unavailable samples and sleep gaps", () => {
    const html = lineChart([point(30, 10), point(28, 20), point(26, null), point(24, 30), point(0, 40)], "cpu", end, 60, "CPU");
    const path = html.match(/<path d="([^"]*)" class="series-primary"/)![1]!;
    expect(path.match(/M/g)?.length).toBe(3);
    expect(path.match(/L/g)?.length).toBe(1);
  });
});
