import { expect, test } from "bun:test";
import { calculatorInputTransitions } from "../src/calculator-navigation";

test("unrelated navigation does not wait for an older calculator transition", async () => {
  const enter = Promise.withResolvers<void>();
  const requests: boolean[] = [];
  const transition = calculatorInputTransitions(active => {
    requests.push(active);
    return active ? enter.promise : Promise.resolve();
  }, () => {});
  expect(transition(false)).toBeUndefined();
  const entering = transition(true);
  const leaving = transition(false);
  expect(transition(false)).toBeUndefined();
  await Promise.resolve();
  expect(requests).toEqual([true]);
  enter.resolve();
  await Promise.all([entering, leaving]);
  expect(requests).toEqual([true, false]);
});

test("re-entry waits for an existing request without queuing a duplicate", async () => {
  const requests: boolean[] = [];
  const transition = calculatorInputTransitions(async active => { requests.push(active); }, () => {});
  const first = transition(true);
  expect(transition(true)).toBe(first);
  await first;
  expect(requests).toEqual([true]);
});
