import type { Connection, Process } from "./mygo";

export type ConnectionSortKey = "process" | "host" | "port" | "protocol" | "state";
export interface ConnectionSort { key: ConnectionSortKey; direction: "ascending" | "descending" }

export function nextConnectionSort(current: ConnectionSort | null, key: ConnectionSortKey): ConnectionSort {
  return { key, direction: current?.key === key && current.direction === "ascending" ? "descending" : "ascending" };
}

export function splitEndpoint(endpoint: string): { host: string; port: string } {
  const separator = endpoint.lastIndexOf(":");
  if (separator < 0) return { host: endpoint, port: "" };
  return { host: endpoint.slice(0, separator).replace(/^\[(.*)\]$/, "$1"), port: endpoint.slice(separator + 1) };
}

export function connectionRows(items: Connection[], processes: Process[], query: string, sort: ConnectionSort | null = null) {
  const byPID = new Map(processes.map(process => [process.pid, process]));
  const search = query.trim().toLowerCase();
  const rows = items.map(connection => {
    const process = byPID.get(connection.pid);
    return { ...connection, process: process?.name || connection.process, path: process?.app || "" };
  }).filter(connection => `${connection.process} ${connection.pid} ${connection.path} ${connection.protocol} ${connection.local} ${connection.remote} ${connection.state}`.toLowerCase().includes(search));
  if (sort) {
    const value = (row: typeof rows[number]) => sort.key === "host" || sort.key === "port"
      ? splitEndpoint(row.remote || row.local)[sort.key]
      : sort.key === "process" ? row.process || String(row.pid) : row[sort.key];
    rows.sort((a, b) => {
      const left = value(a), right = value(b);
      if (!left || !right) return Number(!left) - Number(!right);
      return left.localeCompare(right, undefined, { numeric: true, sensitivity: "base" }) * (sort.direction === "ascending" ? 1 : -1);
    });
  }
  return rows;
}
