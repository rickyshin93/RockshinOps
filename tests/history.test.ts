import { expect, test } from "bun:test";
import { mergeHistory, trendTime } from "../src/history";
import type { Trend } from "../src/mygo";

const point = (time: number) => ({ at: new Date(time).toISOString(), cpu: null } as Trend);
test("incremental history preserves samples, gaps and retention without duplicates or future points", () => {
  const current = [point(0), point(2000), point(4000)];
  const next = [point(4000), point(6000), point(12000)];
  const result = mergeHistory(current, next, 6000, 4);
  expect(result.map(trendTime)).toEqual([2000, 4000, 6000]);
  expect(result[0]).toBe(current[1]);
  expect(result[2]?.cpu).toBeNull();
  expect(current).toHaveLength(3);
  expect(mergeHistory(result, [point(20000)], 20000, 4).map(trendTime)).toEqual([20000]);
});
