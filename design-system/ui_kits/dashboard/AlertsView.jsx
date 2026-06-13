// AlertsView — triage list with severity filter, grouped incidents, and an
// optimistic acknowledge. Color + icon + text for every severity.
function AlertsView({ onOpenServer }) {
  const { Button, Badge } = window.CloudGuardDesignSystem_a66aa9;
  const Icon = window.CGIcon;
  const PageHeader = window.CGPageHeader;
  const [filter, setFilter] = React.useState("all");
  const [acked, setAcked] = React.useState({});
  const all = window.CGData.alerts;
  const rows = filter === "all" ? all : all.filter((a) => a.severity === filter);

  const sevMeta = {
    critical: { variant: "critical", icon: "alert" },
    warning: { variant: "warning", icon: "alert" },
    info: { variant: "muted", icon: "activity" },
  };
  const seg = (val, label) => (
    <button key={val} onClick={() => setFilter(val)} style={{ border: "none", cursor: "pointer", fontFamily: "var(--font-sans)", fontSize: "var(--text-sm)", fontWeight: 500, padding: "6px 12px", borderRadius: "var(--radius-sm)", background: filter === val ? "var(--cg-surface-raised)" : "transparent", color: filter === val ? "var(--cg-text)" : "var(--cg-text-muted)" }}>{label}</button>
  );

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
      <PageHeader title="Alerts & Incidents" description={`${all.filter((a) => !a.acknowledged_at).length} unacknowledged`}
        actions={<Button variant="outline"><Icon name="check" size={16} /> Acknowledge all</Button>} />
      <div style={{ display: "inline-flex", gap: 2, background: "var(--cg-muted)", borderRadius: "var(--radius-md)", padding: 2, alignSelf: "flex-start" }}>
        {seg("all", "All")}{seg("critical", "Critical")}{seg("warning", "Warning")}{seg("info", "Info")}
      </div>
      <div style={{ border: "1px solid var(--cg-border)", borderRadius: "var(--radius-lg)", background: "var(--cg-surface)", overflow: "hidden" }}>
        {rows.map((a, i) => {
          const m = sevMeta[a.severity];
          const isAcked = acked[a.id] || a.acknowledged_at;
          return (
            <div key={a.id} className="cg-trow" style={{ display: "flex", alignItems: "center", gap: 14, padding: "14px 18px", borderTop: i ? "1px solid var(--cg-border)" : "none" }}>
              <span style={{ color: a.severity === "critical" ? "var(--cg-critical)" : a.severity === "warning" ? "var(--cg-warning)" : "var(--cg-text-muted)", display: "flex" }}><Icon name={m.icon} size={18} /></span>
              <Badge variant={m.variant}>{a.severity}</Badge>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: "var(--text-sm)", color: "var(--cg-text)" }}>{a.message}</div>
                <button onClick={() => onOpenServer(window.CGData.servers.find((s) => s.id === a.server_id))} style={{ all: "unset", cursor: "pointer", fontSize: "var(--text-xs)", color: "var(--cg-accent)" }}>{a.server_name}</button>
                <span style={{ fontSize: "var(--text-xs)", color: "var(--cg-text-faint)" }}> · {a.created_at}</span>
              </div>
              {isAcked ? (
                <span style={{ display: "inline-flex", alignItems: "center", gap: 4, fontSize: "var(--text-xs)", color: "var(--cg-success)" }}><Icon name="check" size={14} /> Acknowledged</span>
              ) : (
                <Button variant="ghost" size="sm" onClick={() => setAcked((s) => ({ ...s, [a.id]: true }))}>Acknowledge</Button>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
window.CGAlertsView = AlertsView;
