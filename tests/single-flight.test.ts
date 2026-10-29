import { expect, test } from "bun:test";
import { singleFlight } from "../src/single-flight";

test("foreground navigation joins an unfinished idle preload", async () => {
  let calls = 0;
  const result = Promise.withResolvers<number>();
  const read = singleFlight(() => { calls++; return result.promise; });
  const preload = read();
  const navigation = read();
  expect(navigation).toBe(preload);
  result.resolve(42);
  expect(await navigation).toBe(42);
  expect(await preload).toBe(42);
  expect(calls).toBe(1);
  await read();
  expect(calls).toBe(2);
});

test("failed preload is reported to all waiters and permits a later refresh", async () => {
  let calls = 0;
  const read = singleFlight(async () => {
    if (++calls === 1) throw new Error("collection failed");
    return 7;
  });
  const results = await Promise.allSettled([read(), read()]);
  expect(results.map(result => result.status)).toEqual(["rejected", "rejected"]);
  expect(calls).toBe(1);
  expect(await read()).toBe(7);
});
