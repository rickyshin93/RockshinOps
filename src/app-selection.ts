export function toggleAllAppFiles(paths: string[], allSelected: boolean): string[] {
  return allSelected ? [] : [...paths];
}

export function reconcileAppFiles(paths: string[], selected: string[], selectAll: boolean): string[] {
  if (selectAll) return [...paths];
  const available = new Set(paths);
  return selected.filter(path => available.has(path));
}
