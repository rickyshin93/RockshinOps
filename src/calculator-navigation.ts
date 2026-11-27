// Keep native transitions ordered. Other pages can proceed even when a
// calculator transition from an older navigation is still being restored.
export function calculatorInputTransitions(change: (active: boolean) => Promise<void>, onError: (error: unknown) => void) {
  let requested = false;
  let queue = Promise.resolve();
  return (active: boolean) => {
    if (active === requested) return active ? queue : undefined;
    requested = active;
    const transition = queue.then(() => change(active));
    queue = transition.catch(onError);
    return queue;
  };
}
