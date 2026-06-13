// DashboardView — "is my fleet OK?" KPIs, multi-series fleet chart, resource +
// anomaly panel, and a server summary table. Composes DS primitives.
function PageHeader({ title, description, actions }) {
  return (
    <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: 16, flexWrap: "wrap" }}>
      <div>
        <h1 style={{ margin: 0, fontSize: "var(--text-2xl)", fontWeight: 600, letterSpacing: "var(--tracking-tight)", color: "var(--cg-text)" }}>{title}</h1>
        {description && <p style={{ margin: "4px 0 0", fontSize: "var(--text-sm)", color: "var(--cg-text-muted)" }}>{description}</p>}
      </div>
      {actions && <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>{actions}</div>}
    </div>
  );
}
window.CGPageHeader = PageHeader;

function DashboardView({ onOpenServer }) {
  const { Button, Card, CardHeader, CardTitle, StatCard, ResourceBar } = window.CloudGuardDesignSystem_a66aa9;
  const Icon = window.CGIcon;
  const FleetChart = window.FleetChart;
  const ServerTable = window.CGServerTable;
  const { servers, summary, fleetSeries } = window.CGData;
  const stopped = servers.filter((s) => s.status === "stopped");
  const highDisk = servers.filter((s) => s.disk >= 90);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
      <PageHeader
        title="Welcome back, Admin"
        description={<span>Global uptime <strong style={{ color: "var(--cg-success)", fontWeight: 600 }}>{summary.uptime_percent}%</strong> over the last 24h</span>}
        actions={<>
          <Button><Icon name="plus" size={16} /> Add Server</Button>
          <Button variant="outline"><Icon name="bell" size={16} /> View Alerts</Button>
          <Button variant="ghost" size="icon" aria-label="Refresh"><Icon name="refresh" size={16} /></Button>
        </>}
      />

      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))", gap: 16 }}>
        <StatCard label="Active Servers" value={`${summary.active_servers}/${summary.total_servers}`} icon={<Icon name="server" size={20} />} />
        <StatCard label="Avg CPU Load" value={`${summary.cpu.value}%`} icon={<Icon name="cpu" size={20} />} delta={summary.cpu.delta} deltaSuffix="%" goodDirection="down" />
        <StatCard label="Avg Memory" value={`${summary.memory.value}%`} icon={<Icon name="memory" size={20} />} delta={summary.memory.delta} deltaSuffix="%" goodDirection="down" />
        <StatCard label="Aggregate Network" value="79.8 KB/s" icon={<Icon name="network" size={20} />} />
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "2fr 1fr", gap: 16 }} className="cg-dash-grid">
        <Card>
          <CardHeader><CardTitle style={{ fontSize: "var(--text-base)" }}>Fleet resource trend — CPU %</CardTitle></CardHeader>
          <div style={{ padding: "0 24px 20px" }}><FleetChart series={fleetSeries} /></div>
        </Card>
        <Card>
          <CardHeader><CardTitle style={{ fontSize: "var(--text-base)" }}>Resource Usage</CardTitle></CardHeader>
          <div style={{ padding: "0 24px 24px", display: "flex", flexDirection: "column", gap: 16 }}>
            <ResourceBar label="CPU" value={summary.cpu.value} />
            <ResourceBar label="Memory" value={summary.memory.value} />
            <ResourceBar label="Disk I/O" value={summary.disk.value} />
            <div style={{ display: "flex", justifyContent: "space-between", fontSize: "var(--text-sm)" }}>
              <span style={{ color: "var(--cg-text-muted)" }}>Network</span>
              <span style={{ fontWeight: 500 }}>79.8 KB/s</span>
            </div>
            <div style={{ border: "1px solid var(--cg-border)", borderRadius: "var(--radius-md)", background: "hsl(222 16% 16% / 0.3)", padding: 12 }}>
              <p style={{ margin: "0 0 8px", display: "flex", alignItems: "center", gap: 6, fontSize: "var(--text-xs)", fontWeight: 500, color: "var(--cg-text-muted)" }}>
                <Icon name="activity" size={14} /> Notices
              </p>
              <ul style={{ margin: 0, padding: 0, listStyle: "none", display: "flex", flexDirection: "column", gap: 6 }}>
                {stopped.length > 0 && (
                  <li><button onClick={() => onOpenServer(stopped[0])} style={{ all: "unset", cursor: "pointer", display: "flex", alignItems: "center", gap: 6, fontSize: "var(--text-sm)", color: "var(--cg-critical)" }}>
                    <Icon name="alert" size={14} /> {stopped.length} server{stopped.length > 1 ? "s" : ""} stopped
                  </button></li>
                )}
                {highDisk.map((s) => (
                  <li key={s.id}><button onClick={() => onOpenServer(s)} style={{ all: "unset", cursor: "pointer", display: "flex", alignItems: "center", gap: 6, fontSize: "var(--text-sm)", color: "var(--cg-warning)" }}>
                    <Icon name="alert" size={14} /> Disk &gt; 90% on {s.name}
                  </button></li>
                ))}
              </ul>
            </div>
          </div>
        </Card>
      </div>

      <Card>
        <CardHeader style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between" }}>
          <CardTitle style={{ fontSize: "var(--text-base)" }}>Servers</CardTitle>
          <Button variant="ghost" size="sm">View all</Button>
        </CardHeader>
        <div style={{ paddingBottom: 4 }}>
          <ServerTable servers={servers.slice(0, 6)} onOpenServer={onOpenServer} />
        </div>
      </Card>
    </div>
  );
}

window.CGDashboardView = DashboardView;
