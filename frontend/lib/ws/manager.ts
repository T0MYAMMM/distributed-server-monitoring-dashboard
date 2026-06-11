// DashboardSocket manages the dashboard WebSocket with exponential-backoff
// reconnection. The backend pushes the (IP-masked) server list as a JSON array;
// REST polling in the query hook is the fallback when the socket is down.
import { WS_URL } from "@/config/config";
import type { Server } from "@/lib/api/types";

export interface DashboardSocketHandlers {
  onServers: (servers: Server[]) => void;
  onConnectionChange?: (connected: boolean) => void;
}

export class DashboardSocket {
  private ws: WebSocket | null = null;
  private closed = false;
  private retry = 0;
  private timer: ReturnType<typeof setTimeout> | null = null;

  constructor(private handlers: DashboardSocketHandlers) {}

  connect(): void {
    if (typeof window === "undefined") return;
    this.closed = false;
    this.open();
  }

  private open(): void {
    try {
      this.ws = new WebSocket(WS_URL);
    } catch {
      this.scheduleReconnect();
      return;
    }
    this.ws.onopen = () => {
      this.retry = 0;
      this.handlers.onConnectionChange?.(true);
    };
    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (Array.isArray(data)) this.handlers.onServers(data as Server[]);
      } catch {
        /* ignore malformed frame */
      }
    };
    this.ws.onclose = () => {
      this.handlers.onConnectionChange?.(false);
      this.scheduleReconnect();
    };
    this.ws.onerror = () => this.ws?.close();
  }

  private scheduleReconnect(): void {
    if (this.closed || this.timer) return;
    const delay = Math.min(1000 * 2 ** this.retry, 15000);
    this.retry += 1;
    this.timer = setTimeout(() => {
      this.timer = null;
      if (!this.closed) this.open();
    }, delay);
  }

  close(): void {
    this.closed = true;
    if (this.timer) clearTimeout(this.timer);
    this.timer = null;
    this.ws?.close();
    this.ws = null;
  }
}
