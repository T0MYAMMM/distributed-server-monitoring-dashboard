// API DTOs mirroring the Go backend's JSON contract (internal/domain). Keep the
// snake_case field names exactly as the backend serializes them.

export type ServerStatus = "running" | "stopped" | "maintenance";

export interface Server {
  id: string;
  name: string;
  type: string;
  location: string;
  ip_address: string;
  hostname: string;
  tailscale_ip: string;
  status: ServerStatus;
  uptime: number;
  network_in: number;
  network_out: number;
  cpu: number;
  memory: number;
  disk: number;
  os_type: string;
  cpu_info: string;
  total_memory: number;
  total_disk: number;
  order_index: number;
  first_seen: string;
  last_update: string;
}

export interface Client {
  name: string;
  created_at: string;
}

export interface MetricSample {
  ts: number; // unix seconds
  cpu: number;
  memory: number;
  disk: number;
  network_in: number;
  network_out: number;
}

export interface FleetMetric {
  value: number;
  delta: number;
}

export interface FleetSummary {
  range_seconds: number;
  active_servers: number;
  total_servers: number;
  cpu: FleetMetric;
  memory: FleetMetric;
  disk: FleetMetric;
  network: FleetMetric;
  uptime_percent: number;
}

export type AlertSeverity = "info" | "warning" | "critical";

export interface Alert {
  id: number;
  type: string;
  server_id: string;
  server_name: string;
  severity: AlertSeverity;
  message: string;
  created_at: string;
  acknowledged_at: string;
}

export interface UnknownAgent {
  name: string;
  remote_addr: string;
  last_seen: string;
  count: number;
}

export type MetricRange = "1h" | "6h" | "24h" | "7d";

export type LogLevel = "DEBUG" | "INFO" | "WARN" | "ERROR";

export interface LogLine {
  id: number;
  server: string;
  ts: string;
  level: string;
  module: string;
  message: string;
  source_file: string;
}
