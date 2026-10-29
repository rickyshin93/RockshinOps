// Idle preload and foreground refresh share a request until it settles.
export function singleFlight<T>(read: () => Promise<T>): () => Promise<T> {
  let pending: Promise<T> | undefined;
  return () => pending ??= Promise.resolve().then(read).finally(() => { pending = undefined; });
}
