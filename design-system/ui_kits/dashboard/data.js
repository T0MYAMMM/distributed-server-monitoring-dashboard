// Fake fleet data for the CloudGuard dashboard UI kit. Shapes mirror the
// backend DTOs (lib/api/types.ts): Server, FleetSummary, Alert, LogLine.
(function () {
  function series(base, jitter, n, drift) {
    const out = [];
    let v = base;
    for (let i = 0; i < n; i++) {
      v += (Math.random() - 0.5) * jitter + (drift || 0);
      out.push(Math.max(1, Math.min(99, Math.round(v * 10) / 10)));
    }
    return out;
  }

  const servers = [
    { id: "a1", name: "web-1", type: "VPS", location: "US", status: "running", uptime: 1641600, cpu: 34, memory: 61, disk: 48, network_in: 9420, network_out: 1740, os_type: "debian 12", cpu_info: "AMD EPYC 7402P · 4 vCPU", total_memory: 8, total_disk: 80, hostname: "web-1.tail9c2.ts.net", tailscale_ip: "100.92.14.3", ip_address: "203.0.113.7", spark: series(32, 14, 24) },
    { id: "a2", name: "db-1", type: "Dedicated", location: "DE", status: "running", uptime: 1641600, cpu: 22, memory: 78, disk: 91, network_in: 1200, network_out: 506, os_type: "ubuntu 22.04", cpu_info: "Intel Xeon E-2386G · 12 threads", total_memory: 64, total_disk: 960, hostname: "ks-le-b.tail9c2.ts.net", tailscale_ip: "100.92.14.9", ip_address: "198.51.100.21", spark: series(72, 8, 24, 0.4) },
    { id: "a3", name: "cache-2", type: "VPS", location: "US", status: "running", uptime: 86400, cpu: 12, memory: 30, disk: 22, network_in: 19500, network_out: 4200, os_type: "debian 12", cpu_info: "AMD EPYC 7282 · 2 vCPU", total_memory: 4, total_disk: 40, hostname: "cache-2.tail9c2.ts.net", tailscale_ip: "100.92.14.4", ip_address: "203.0.113.44", spark: series(14, 10, 24) },
    { id: "a4", name: "edge-fra", type: "VPS", location: "DE", status: "maintenance", uptime: 0, cpu: 0, memory: 0, disk: 0, network_in: 0, network_out: 0, os_type: "alpine 3.20", cpu_info: "—", total_memory: 2, total_disk: 20, hostname: "edge-fra.tail9c2.ts.net", tailscale_ip: "100.92.14.12", ip_address: "—", spark: [] },
    { id: "a5", name: "scraper-3", type: "VPS", location: "NL", status: "running", uptime: 432000, cpu: 67, memory: 54, disk: 71, network_in: 8800, network_out: 12400, os_type: "ubuntu 24.04", cpu_info: "Ampere Altra · 4 vCPU (arm64)", total_memory: 8, total_disk: 60, hostname: "scraper-3.tail9c2.ts.net", tailscale_ip: "100.92.14.18", ip_address: "192.0.2.88", spark: series(60, 16, 24) },
    { id: "a6", name: "backup-old", type: "Dedicated", location: "FR", status: "stopped", uptime: 0, cpu: 0, memory: 0, disk: 88, network_in: 0, network_out: 0, os_type: "debian 11", cpu_info: "Intel Atom C3558 · 4 cores", total_memory: 16, total_disk: 2000, hostname: "backup-old.tail9c2.ts.net", tailscale_ip: "100.92.14.31", ip_address: "203.0.113.99", spark: [] },
    { id: "a7", name: "ci-runner", type: "VPS", location: "US", status: "running", uptime: 259200, cpu: 88, memory: 72, disk: 40, network_in: 3400, network_out: 2900, os_type: "ubuntu 22.04", cpu_info: "AMD EPYC 7763 · 8 vCPU", total_memory: 16, total_disk: 160, hostname: "ci-runner.tail9c2.ts.net", tailscale_ip: "100.92.14.22", ip_address: "198.51.100.77", spark: series(80, 12, 24, 0.3) },
    { id: "a8", name: "home-nas", type: "Dedicated", location: "US", status: "running", uptime: 7776000, cpu: 8, memory: 41, disk: 63, network_in: 540, network_out: 320, os_type: "TrueNAS 24", cpu_info: "Intel i5-9400 · 6 cores", total_memory: 32, total_disk: 8000, hostname: "home-nas.tail9c2.ts.net", tailscale_ip: "100.92.14.40", ip_address: "—", spark: series(9, 6, 24) },
    { id: "a9", name: "vpn-gw", type: "VPS", location: "SG", status: "running", uptime: 5184000, cpu: 19, memory: 33, disk: 28, network_in: 24800, network_out: 22100, os_type: "debian 12", cpu_info: "Ampere Altra · 2 vCPU (arm64)", total_memory: 4, total_disk: 40, hostname: "vpn-gw.tail9c2.ts.net", tailscale_ip: "100.92.14.51", ip_address: "203.0.113.150", spark: series(20, 9, 24) },
  ];

  const summary = {
    active_servers: servers.filter((s) => s.status === "running").length,
    total_servers: servers.length,
    cpu: { value: 35.1, delta: -4.2 },
    memory: { value: 49.1, delta: 2.8 },
    disk: { value: 53.3, delta: 1.1 },
    network: { value: 81720, delta: 0 },
    uptime_percent: 99.98,
  };

  // Fleet time-series: one line per running server (multi-series chart).
  const fleetSeries = servers
    .filter((s) => s.status === "running")
    .slice(0, 6)
    .map((s, i) => ({ id: s.id, name: s.name, color: `var(--cg-series-${i + 1})`, data: s.spark.length ? s.spark : series(30, 12, 24) }));

  const alerts = [
    { id: 5, type: "threshold", server_id: "a2", server_name: "db-1", severity: "critical", message: "Disk usage 91% exceeds threshold (90%)", created_at: "2m ago", acknowledged_at: "" },
    { id: 4, type: "status", server_id: "a6", server_name: "backup-old", severity: "critical", message: "Server transitioned running → stopped", created_at: "14m ago", acknowledged_at: "" },
    { id: 3, type: "threshold", server_id: "a7", server_name: "ci-runner", severity: "warning", message: "CPU 88% sustained over 10m", created_at: "31m ago", acknowledged_at: "" },
    { id: 2, type: "status", server_id: "a4", server_name: "edge-fra", severity: "info", message: "Server entered maintenance", created_at: "1h ago", acknowledged_at: "" },
    { id: 1, type: "threshold", server_id: "a5", server_name: "scraper-3", severity: "warning", message: "Disk 71% trending up", created_at: "3h ago", acknowledged_at: "9:14 AM" },
  ];

  const modules = ["app-api", "app-web", "nginx", "postgres", "scheduler", "claude-agents-worker"];
  const VERBS = ["GET", "POST", "PUT", "DELETE"];
  const URLS = ["/api/v1/servers", "/api/v1/metrics/summary", "/login", "/healthz", "/api/v1/logs/stream", "/api/v1/alerts"];
  const STATUS = [200, 200, 200, 201, 204, 301, 401, 429, 500, 503];
  function pad(n) { return String(n).padStart(2, "0"); }
  const logs = [];
  for (let i = 0; i < 60; i++) {
    const r = Math.random();
    const level = r > 0.93 ? "ERROR" : r > 0.82 ? "WARN" : r > 0.2 ? "INFO" : "DEBUG";
    const code = STATUS[Math.floor(Math.random() * STATUS.length)];
    const verb = VERBS[Math.floor(Math.random() * VERBS.length)];
    const url = URLS[Math.floor(Math.random() * URLS.length)];
    const dur = Math.floor(Math.random() * 240);
    logs.push({
      id: i + 1,
      ts: `14:${pad(2 + Math.floor(i / 12))}:${pad((i * 7) % 60)}.${pad(i * 13 % 100)}`,
      level,
      module: modules[Math.floor(Math.random() * modules.length)],
      verb, url, code, dur,
      msg: level === "DEBUG" ? `cache hit key=srv:${i}` : level === "ERROR" ? `upstream timeout after ${dur}ms` : `${verb} ${url} ${code}`,
    });
  }

  // Volume histogram by level over the last 24 buckets
  const histogram = Array.from({ length: 24 }, () => ({
    info: Math.floor(Math.random() * 40) + 10,
    warn: Math.floor(Math.random() * 8),
    error: Math.floor(Math.random() * 4),
  }));

  window.CGData = { servers, summary, fleetSeries, alerts, modules, logs, histogram };
})();
