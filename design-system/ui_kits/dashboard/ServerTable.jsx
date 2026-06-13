// ServerTable — fleet table composing StatusBadge + ResourceBar. Row click opens
// the server detail drawer. Used on Dashboard (top N) and Servers (full).
function flag(code) {
  if (!code || code.length !== 2 || !/^[a-zA-Z]{2}$/.test(code)) return "🌐";
  const base = 0x1f1e6;
  const u = code.toUpperCase();
  return String.fromCodePoint(base + u.charCodeAt(0) - 65, base + u.charCodeAt(1) - 65);
}
function fmtUptime(s) {
  if (!s) return "—";
  const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600);
  return d > 0 ? `${d}d ${h}h` : `${h}h`;
}
function fmtRate(b) {
  if (!b) return "0 B/s";
  const u = ["B/s", "KB/s", "MB/s"]; let v = b, i = 0;
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++; }
  return `${v.toFixed(v >= 10 || i === 0 ? 0 : 1)} ${u[i]}`;
}
window.CGfmt = { flag, fmtUptime, fmtRate };

function ServerTable({ servers, onOpenServer }) {
  const { StatusBadge, ResourceBar, Badge } = window.CloudGuardDesignSystem_a66aa9;
  const Icon = window.CGIcon;
  const th = { textAlign: "left", padding: "10px 16px", fontSize: "var(--text-xs)", fontWeight: 500, color: "var(--cg-text-muted)", whiteSpace: "nowrap", textTransform: "none" };
  const td = { padding: "var(--row-pad-y) 16px", borderTop: "1px solid var(--cg-border)", fontSize: "var(--text-sm)", color: "var(--cg-text)", verticalAlign: "middle" };
  return (
    <div style={{ overflowX: "auto" }}>
      <table style={{ width: "100%", borderCollapse: "collapse" }}>
        <thead>
          <tr>
            <th style={th}>Status</th>
            <th style={th}>Server</th>
            <th style={th}>Location</th>
            <th style={{ ...th, minWidth: 120 }}>CPU</th>
            <th style={{ ...th, minWidth: 120 }}>Memory</th>
            <th style={{ ...th, minWidth: 120 }}>Disk</th>
            <th style={th}>Network</th>
            <th style={th}>Uptime</th>
            <th style={{ ...th, width: 32 }}></th>
          </tr>
        </thead>
        <tbody>
          {servers.map((s) => (
            <tr key={s.id} className="cg-trow" style={{ cursor: "pointer" }} onClick={() => onOpenServer(s)}>
              <td style={td}><StatusBadge status={s.status} /></td>
              <td style={td}>
                <div style={{ fontWeight: 500 }}>{s.name}</div>
                <div style={{ fontSize: "var(--text-xs)", color: "var(--cg-text-muted)" }}>{s.os_type} · {s.type}</div>
              </td>
              <td style={{ ...td, whiteSpace: "nowrap" }}><span style={{ marginRight: 6 }}>{flag(s.location)}</span><span style={{ color: "var(--cg-text-muted)" }}>{s.location}</span></td>
              <td style={td}><ResourceBar value={s.cpu} /></td>
              <td style={td}><ResourceBar value={s.memory} /></td>
              <td style={td}><ResourceBar value={s.disk} /></td>
              <td style={{ ...td, whiteSpace: "nowrap", fontSize: "var(--text-xs)", color: "var(--cg-text-muted)" }}>
                <div style={{ display: "flex", alignItems: "center", gap: 4 }}><Icon name="down" size={12} /> {fmtRate(s.network_in)}</div>
                <div style={{ display: "flex", alignItems: "center", gap: 4 }}><Icon name="up" size={12} /> {fmtRate(s.network_out)}</div>
              </td>
              <td style={{ ...td, whiteSpace: "nowrap", fontVariantNumeric: "tabular-nums" }}>{fmtUptime(s.uptime)}</td>
              <td style={{ ...td, color: "var(--cg-text-faint)" }}><Icon name="chevronRight" size={16} /></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

window.CGServerTable = ServerTable;
