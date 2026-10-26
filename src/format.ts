import { t, i18n } from "./i18n";
export function capacityParts(bytes: number): { value: string; unit: string } {
  if (bytes >= 1_000_000_000_000) return { value: (bytes / 1_000_000_000_000).toFixed(2), unit: "TB" };
  if (bytes > 0 && bytes < 1_000_000_000) return { value: (bytes / 1_000_000).toFixed(1), unit: "MB" };
  return { value: (bytes / 1_000_000_000).toFixed(1), unit: "GB" };
}

export function formatCapacity(bytes: number): string {
  const { value, unit } = capacityParts(bytes);
  return `${value} ${unit}`;
}

export function memoryParts(bytes: number): { value: string; unit: string } {
  if (bytes >= 1024 ** 3) return { value: (bytes / 1024 ** 3).toFixed(1), unit: "GiB" };
  return { value: (bytes / 1024 ** 2).toFixed(1), unit: "MiB" };
}
export function formatMemory(bytes: number): string { const p = memoryParts(bytes); return `${p.value} ${p.unit}`; }
export function rateParts(bytes: number): { value: string; unit: string } {
  if (bytes >= 1e9) return { value: (bytes / 1e9).toFixed(2), unit: "GB/s" };
  if (bytes >= 1e6) return { value: (bytes / 1e6).toFixed(1), unit: "MB/s" };
  if (bytes >= 1e3) return { value: (bytes / 1e3).toFixed(1), unit: "KB/s" };
  return { value: bytes.toFixed(0), unit: "B/s" };
}
export function formatRate(bytes: number): string { const p = rateParts(bytes); return `${p.value} ${p.unit}`; }
export function percent(value: number): string { return `${value.toFixed(value >= 10 ? 0 : 1)}%`; }
export function duration(seconds: number): string {
  if (seconds < 0) return t("不可用");
  if (seconds >= 86400) return `${Math.floor(seconds / 86400)} ${t("天")} ${Math.floor(seconds % 86400 / 3600)} ${t("小时")}`;
  if (seconds >= 3600) return `${Math.floor(seconds / 3600)} ${t("小时")} ${Math.floor(seconds % 3600 / 60)} ${t("分钟")}`;
  return `${Math.floor(seconds / 60)} ${t("分钟")}`;
}
export function timeLabel(time: string): string { return new Date(time).toLocaleTimeString(i18n.language, { hour12: false }); }
export function escapeHTML(value: string | number): string {
  return String(value).replace(/[&<>"']/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]!);
}
