// ServersView — full fleet table with filters, export, density toggle, plus the
// net-new Server Detail drawer (per §4.2: status, IPs, charts, host alerts/logs).
function ServerDetail({ server, onClose, onViewLogs }) {
  const { Button, Badge, StatusBadge, ResourceBar, Sparkline, Card } = window.CloudGuardDesignSystem_a66aa9;
  const Icon = window.CGIcon;
  const { flag } = window.CGfmt;
  if (!server) return null;
  const hostAlerts = window.CGData.alerts.filter((a) => a.server_id === server.id);
  const meta = [
    ["Hostname", server.hostname], ["OS", server.os_type], ["CPU", server.cpu_info], ["Type", server.type],
    ["Total memory", `${server.total_memory} GB`], ["Total disk", `${server.total_disk} GB`],
    ["Tailscale IP", server.tailscale_ip], ["Public IP", server.ip_address],
  ];
  return (
    <div onClick={onClose} style={{ position: "absolute", inset: 0, background: "hsl(222 40% 2% / 0.55)", zIndex: 40, display: "flex", justifyContent: "flex-end" }}>
      <div onClick={(e) => e.stopPropagation()} style={{ width: 480, maxWidth: "92%", height: "100%", background: "var(--cg-surface)", borderLeft: "1px solid var(--cg-border)", boxShadow: "var(--shadow-lg)", overflowY: "auto", display: "flex", flexDirection: "column" }}>
        <div style={{ padding: "20px 24px", borderBottom: "1px solid var(--cg-border)", display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: 12 }}>
          <div>
            <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
              <h2 style={{ margin: 0, fontSize: "var(--text-xl)", fontWeight: 600, letterSpacing: "var(--tracking-tight)", color: "var(--cg-text)" }}>{server.name}</h2>
              <StatusBadge status={server.status} />
            </div>
            <p style={{ margin: "4px 0 0", fontSize: "var(--text-sm)", color: "var(--cg-text-muted)" }}>{flag(server.location)} {server.location} · {server.os_type}</p>
          </div>
          <button onClick={onClose} aria-label="Close" style={{ background: "none", border: "none", cursor: "pointer", color: "var(--cg-text-muted)", fontSize: 22, lineHeight: 1, padding: 4 }}>×</button>
        </div>
        <div style={{ padding: 24, display: "flex", flexDirection: "column", gap: 20 }}>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr 1fr", gap: 12 }}>
            {[["CPU", server.cpu], ["Memory", server.memory], ["Disk", server.disk]].map(([l, v]) => (
              <div key={l} style={{ border: "1px solid var(--cg-border)", borderRadius: "var(--radius-md)", padding: 12 }}>
                <div style={{ fontSize: "var(--text-xs)", color: "var(--cg-text-muted)", marginBottom: 6 }}>{l}</div>
                <div style={{ fontSize: "var(--text-lg)", fontWeight: 600, fontVariantNumeric: "tabular-nums" }}>{v}%</div>
                <div style={{ marginTop: 6 }}><ResourceBar value={v} showValue={false} /></div>
              </div>
            ))}
          </div>
          {server.spark.length > 0 && (
            <div style={{ border: "1px solid var(--cg-border)", borderRadius: "var(--radius-md)", padding: 16 }}>
              <div style={{ fontSize: "var(--text-xs)", color: "var(--cg-text-muted)", marginBottom: 8 }}>CPU trend · last 24h</div>
              <Sparkline data={server.spark} width={420} height={56} />
            </div>
          )}
          <div>
            <h3 style={{ margin: "0 0 10px", fontSize: "var(--text-2xs)", textTransform: "uppercase", letterSpacing: "var(--tracking-wide)", color: "var(--cg-text-faint)" }}>System</h3>
            <dl style={{ margin: 0, display: "grid", gridTemplateColumns: "1fr 1fr", gap: "12px 16px" }}>
              {meta.map(([k, v]) => (
                <div key={k}>
                  <dt style={{ fontSize: "var(--text-xs)", color: "var(--cg-text-muted)" }}>{k}</dt>
                  <dd style={{ margin: "2px 0 0", fontSize: "var(--text-sm)", fontFamily: k.includes("IP") || k === "Hostname" ? "var(--font-mono)" : "inherit", color: "var(--cg-text)", overflowWrap: "anywhere" }}>{v || "—"}</dd>
                </div>
              ))}
            </dl>
          </div>
          <div>
            <h3 style={{ margin: "0 0 10px", fontSize: "var(--text-2xs)", textTransform: "uppercase", letterSpacing: "var(--tracking-wide)", color: "var(--cg-text-faint)" }}>Recent alerts</h3>
            {hostAlerts.length === 0 ? (
              <p style={{ margin: 0, fontSize: "var(--text-sm)", color: "var(--cg-text-muted)" }}>No alerts for this host.</p>
            ) : hostAlerts.map((a) => (
              <div key={a.id} style={{ display: "flex", gap: 8, alignItems: "center", padding: "8px 0", borderTop: "1px solid var(--cg-border)" }}>
                <Badge variant={a.severity === "critical" ? "critical" : a.severity === "warning" ? "warning" : "muted"}>{a.severity}</Badge>
                <span style={{ fontSize: "var(--text-sm)", flex: 1 }}>{a.message}</span>
                <span style={{ fontSize: "var(--text-xs)", color: "var(--cg-text-faint)", whiteSpace: "nowrap" }}>{a.created_at}</span>
              </div>
            ))}
          </div>
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap", borderTop: "1px solid var(--cg-border)", paddingTop: 16 }}>
            <Button variant="outline" size="sm" onClick={() => onViewLogs(server)}><Icon name="logs" size={16} /> Logs for this host</Button>
            <Button variant="outline" size="sm"><Icon name="clock" size={16} /> Force status</Button>
            <Button variant="destructive" size="sm">Delete</Button>
          </div>
        </div>
      </div>
    </div>
  );
}
window.CGServerDetail = ServerDetail;

function ServersView({ onOpenServer }) {
  const { Button, Input, Badge } = window.CloudGuardDesignSystem_a66aa9;
  const Icon = window.CGIcon;
  const ServerTable = window.CGServerTable;
  const PageHeader = window.CGPageHeader;
  const [q, setQ] = React.useState("");
  const [status, setStatus] = React.useState("all");
  const [compact, setCompact] = React.useState(false);
  let rows = window.CGData.servers;
  if (status !== "all") rows = rows.filter((s) => s.status === status);
  if (q) rows = rows.filter((s) => s.name.toLowerCase().includes(q.toLowerCase()));

  const seg = (val, label) => (
    <button onClick={() => setStatus(val)} style={{ border: "none", cursor: "pointer", fontFamily: "var(--font-sans)", fontSize: "var(--text-sm)", fontWeight: 500, padding: "6px 12px", borderRadius: "var(--radius-sm)", background: status === val ? "var(--cg-surface-raised)" : "transparent", color: status === val ? "var(--cg-text)" : "var(--cg-text-muted)" }}>{label}</button>
  );

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
      <PageHeader title="Servers" description={`${window.CGData.servers.length} nodes across your tailnet`}
        actions={<>
          <Button variant="outline"><Icon name="download" size={16} /> Export CSV</Button>
          <Button><Icon name="plus" size={16} /> Add Server</Button>
        </>} />
      <div style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" }}>
        <div style={{ display: "inline-flex", gap: 2, background: "var(--cg-muted)", borderRadius: "var(--radius-md)", padding: 2 }}>
          {seg("all", "All")}{seg("running", "Running")}{seg("stopped", "Stopped")}{seg("maintenance", "Pending")}
        </div>
        <div style={{ position: "relative", minWidth: 200, flex: "0 1 280px" }}>
          <span style={{ position: "absolute", left: 10, top: "50%", transform: "translateY(-50%)", color: "var(--cg-text-muted)", display: "flex" }}><Icon name="search" size={16} /></span>
          <Input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Filter by name…" style={{ paddingLeft: 32 }} />
        </div>
        <button onClick={() => setCompact((v) => !v)} style={{ marginLeft: "auto", display: "inline-flex", alignItems: "center", gap: 6, height: 40, padding: "0 12px", borderRadius: "var(--radius-md)", border: "1px solid var(--cg-border)", background: compact ? "var(--cg-secondary)" : "transparent", color: "var(--cg-text)", cursor: "pointer", fontSize: "var(--text-sm)", fontFamily: "var(--font-sans)" }}>
          <Icon name="filter" size={16} /> {compact ? "Compact" : "Comfortable"}
        </button>
      </div>
      <div className={compact ? "density-compact" : ""} style={{ border: "1px solid var(--cg-border)", borderRadius: "var(--radius-lg)", background: "var(--cg-surface)", overflow: "hidden" }}>
        <ServerTable servers={rows} onOpenServer={onOpenServer} />
      </div>
    </div>
  );
}
window.CGServersView = ServersView;
