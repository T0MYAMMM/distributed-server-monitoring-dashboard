// Typed API client: one function per endpoint, targeting the canonical
// /api/v1 surface. Every function returns parsed, typed data or throws ApiError.
import { API_URL } from "@/config/config";
import { getToken } from "@/lib/auth";
import type {
  Alert,
  Client,
  FleetSummary,
  LogLine,
  MetricRange,
  MetricSample,
  Server,
  ServerStatus,
  UnknownAgent,
} from "./types";

const V1 = `${API_URL}/api/v1`;

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = "ApiError";
  }
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  auth?: boolean;
}

async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = {};
  if (opts.body !== undefined) headers["Content-Type"] = "application/json";
  const token = getToken();
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(`${V1}${path}`, {
    method: opts.method ?? "GET",
    headers,
    body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
  });

  if (!res.ok) {
    let message = res.statusText;
    try {
      const data = await res.json();
      // Accept both the flat {error:"msg"} and nested {error:{message}} shapes.
      message =
        typeof data.error === "string"
          ? data.error
          : data.error?.message ?? message;
    } catch {
      /* non-JSON error body */
    }
    throw new ApiError(res.status, message);
  }

  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

// --- servers ---
export const getServers = () => request<Server[]>("/servers");
export const getServer = (id: string) => request<Server>(`/servers/${id}`);
export const deleteServer = (id: string) =>
  request<void>(`/servers/${id}`, { method: "DELETE" });
export const setServerStatus = (id: string, status: ServerStatus) =>
  request<Server>(`/servers/${id}/status`, { method: "PUT", body: { status } });
export const setServerOrder = (id: string, order_index: number) =>
  request<void>(`/servers/${id}/order`, { method: "PUT", body: { order_index } });

// --- clients ---
export const getClients = () => request<Client[]>("/clients");
export const addClient = (name: string) =>
  request<void>("/clients", { method: "POST", body: { name } });

// --- metrics ---
export const getServerMetrics = (id: string, range: MetricRange) =>
  request<MetricSample[]>(`/servers/${id}/metrics?range=${range}`);
export const getFleetSummary = (range: MetricRange) =>
  request<FleetSummary>(`/metrics/summary?range=${range}`);

// --- alerts ---
export const getAlerts = (severity?: string, limit?: number) => {
  const params = new URLSearchParams();
  if (severity) params.set("severity", severity);
  if (limit) params.set("limit", String(limit));
  const qs = params.toString();
  return request<Alert[]>(`/alerts${qs ? `?${qs}` : ""}`);
};
export const acknowledgeAlert = (id: number) =>
  request<void>(`/alerts/${id}/acknowledge`, { method: "POST" });

// --- logs ---
export interface LogFilter {
  level?: string;
  q?: string;
  since?: string;
  limit?: number;
}
export const getServerLogs = (id: string, opts: LogFilter = {}) => {
  const params = new URLSearchParams();
  if (opts.level) params.set("level", opts.level);
  if (opts.q) params.set("q", opts.q);
  if (opts.since) params.set("since", opts.since);
  if (opts.limit) params.set("limit", String(opts.limit));
  const qs = params.toString();
  return request<LogLine[]>(`/servers/${id}/logs${qs ? `?${qs}` : ""}`);
};

// logsStreamUrl is the SSE endpoint for live-tailing a server's logs.
export const logsStreamUrl = (id: string, after: number) =>
  `${V1}/servers/${id}/logs/stream?after=${after}`;

// --- admin ---
export const getUnknownAgents = () =>
  request<UnknownAgent[]>("/admin/unknown-agents");

// --- auth ---
export const getAuthStatus = () =>
  request<{ initialized: boolean }>("/auth/status");
export const login = (password: string) =>
  request<{ token: string }>("/auth/login", { method: "POST", body: { password } });
export const initialize = (password: string) =>
  request<{ success: boolean }>("/auth/initialize", { method: "POST", body: { password } });
export const resetPassword = (oldPassword: string, newPassword: string) =>
  request<{ success: boolean }>("/auth/reset-password", {
    method: "POST",
    body: { oldPassword, newPassword },
  });
