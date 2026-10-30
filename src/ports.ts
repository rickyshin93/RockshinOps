import type { Connection, Process } from "./mygo";
import { splitEndpoint } from "./connections";

export type PortScope = "all" | "loopback" | "wildcard";
export interface ListeningPort {
  port: number;
  pid: number;
  process: string;
  app: string;
  identity: string;
  addresses: string[];
  scopes: Set<PortScope | "interface">;
}

export function addressScope(host: string): PortScope | "interface" {
  if (host === "*" || host === "0.0.0.0" || host === "::") return "wildcard";
  if (host === "::1" || /^127\./.test(host) || /^::ffff:127\./i.test(host)) return "loopback";
  return "interface";
}

// Preserve multiple owners and bindings; neither implies a port conflict.
export function listeningPorts(items: Connection[], processes: Process[]): ListeningPort[] {
  const byPID = new Map(processes.map(process => [process.pid, process]));
  const rows = new Map<string, ListeningPort>();
  for (const connection of items) {
    if (connection.protocol !== "TCP" || connection.state !== "LISTEN") continue;
    const { host, port: text } = splitEndpoint(connection.local);
    const port = Number(text);
    if (!Number.isInteger(port) || port < 1 || port > 65535) continue;
    const key = `${port}:${connection.pid}`;
    let row = rows.get(key);
    if (!row) {
      const process = byPID.get(connection.pid);
      row = { port, pid: connection.pid, process: process?.name || connection.process, app: process?.app || "", identity: process?.identity || "", addresses: [], scopes: new Set() };
      rows.set(key, row);
    }
    const address = host === "*" && connection.family === "IPv4" ? `0.0.0.0:${port}`
      : host === "*" && connection.family === "IPv6" ? `[::]:${port}` : connection.local;
    if (!row.addresses.includes(address)) row.addresses.push(address);
    row.scopes.add(addressScope(host));
  }
  return [...rows.values()].sort((a, b) => a.port - b.port || a.pid - b.pid);
}

export function filterPorts(rows: ListeningPort[], query: string, scope: PortScope): ListeningPort[] {
  const search = query.trim().toLowerCase();
  return rows.filter(row => (scope === "all" || row.scopes.has(scope)) && (!search || (/^:\d+$/.test(search) ? row.port === Number(search.slice(1)) : /^\d+$/.test(search)
    ? row.port === Number(search) || row.pid === Number(search)
    : `${row.process} ${row.app} ${row.addresses.join(" ")}`.toLowerCase().includes(search))));
}
