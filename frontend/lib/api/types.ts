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

// --- settings ---
export type SettingKind = "string" | "int" | "float" | "bool" | "enum";

export interface SettingField {
  key: string;
  section: string;
  label: string;
  help?: string;
  kind: SettingKind;
  value: string;
  default: string;
  options?: string[];
  env_var?: string;
  env_locked: boolean;
  restart_required?: boolean;
  min?: number;
  max?: number;
}

export interface SettingsDoc {
  fields: SettingField[];
  about: Record<string, string>;
}

// --- notification channels ---
export type ChannelType = "slack" | "discord" | "ntfy" | "webhook" | "pagerduty" | "email";

export interface NotificationChannel {
  id: number;
  type: ChannelType;
  name: string;
  config: Record<string, string>;
  enabled: boolean;
  secrets_set?: string[];
  last_status: "" | "ok" | "error";
  last_error: string;
  last_delivery: string;
  created_at: string;
}

// --- feedback ---
export type FeedbackCategory = "bug" | "idea" | "praise" | "general";

export interface Feedback {
  id: number;
  category: FeedbackCategory;
  message: string;
  page: string;
  created_at: string;
}

// --- analytics ---
export type AnalyticsRange = "24h" | "7d" | "30d" | "90d";

export interface ServerStat {
  id: string;
  name: string;
  status: ServerStatus;
  cpu: number;
  memory: number;
  disk: number;
  uptime_percent: number;
  disk_days_to_full: number; // -1 = no upward trend
}

export interface LogVolumePoint {
  ts: number;
  debug: number;
  info: number;
  warn: number;
  error: number;
}

export interface ModuleStat {
  module: string;
  total: number;
  errors: number;
}
