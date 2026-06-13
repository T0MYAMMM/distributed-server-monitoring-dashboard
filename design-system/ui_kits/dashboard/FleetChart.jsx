// FleetChart — a themed multi-series line chart (one line per server), with a
// range selector and legend. Pure SVG so it has no chart-lib dependency.
function FleetChart({ series = [], height = 240 }) {
  const [range, setRange] = React.useState("24h");
  const [hover, setHover] = React.useState(null);
  const W = 760, H = height, padL = 4, padR = 4, padT = 12, padB = 18;
  const n = series[0]?.data.length || 0;
  const all = series.flatMap((s) => s.data);
  const max = Math.max(100, ...all);
  const min = 0;
  const xAt = (i) => padL + (i / (n - 1)) * (W - padL - padR);
  const yAt = (v) => padT + (1 - (v - min) / (max - min)) * (H - padT - padB);

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 10, flexWrap: "wrap", gap: 8 }}>
        <div style={{ display: "flex", flexWrap: "wrap", gap: 14 }}>
          {series.map((s) => (
            <span key={s.id} style={{ display: "inline-flex", alignItems: "center", gap: 6, fontSize: "var(--text-xs)", color: "var(--cg-text-muted)" }}>
              <span style={{ width: 10, height: 3, borderRadius: 2, background: s.color }} />
              {s.name}
            </span>
          ))}
        </div>
        <div style={{ display: "inline-flex", gap: 2, background: "var(--cg-muted)", borderRadius: "var(--radius-md)", padding: 2 }}>
          {["1h", "6h", "24h", "7d"].map((r) => (
            <button key={r} onClick={() => setRange(r)}
              style={{
                border: "none", cursor: "pointer", fontFamily: "var(--font-sans)",
                fontSize: "var(--text-xs)", fontWeight: 500, padding: "4px 10px", borderRadius: "var(--radius-sm)",
                background: range === r ? "var(--cg-surface-raised)" : "transparent",
                color: range === r ? "var(--cg-text)" : "var(--cg-text-muted)",
              }}>
              {r}
            </button>
          ))}
        </div>
      </div>
      <svg viewBox={`0 0 ${W} ${H}`} width="100%" height={H} style={{ overflow: "visible" }}
        onMouseLeave={() => setHover(null)}
        onMouseMove={(e) => {
          const rect = e.currentTarget.getBoundingClientRect();
          const x = ((e.clientX - rect.left) / rect.width) * W;
          const i = Math.round(((x - padL) / (W - padL - padR)) * (n - 1));
          setHover(Math.max(0, Math.min(n - 1, i)));
        }}>
        {[0, 25, 50, 75, 100].map((g) => (
          <g key={g}>
            <line x1={padL} x2={W - padR} y1={yAt(g)} y2={yAt(g)} stroke="var(--cg-border)" strokeWidth="1" strokeDasharray={g === 0 ? "0" : "2 4"} />
            <text x={W - padR} y={yAt(g) - 3} textAnchor="end" fontSize="9" fontFamily="var(--font-mono)" fill="var(--cg-text-faint)">{g}</text>
          </g>
        ))}
        {series.map((s) => {
          const d = s.data.map((v, i) => `${i === 0 ? "M" : "L"}${xAt(i).toFixed(1)} ${yAt(v).toFixed(1)}`).join(" ");
          return <path key={s.id} d={d} fill="none" stroke={s.color} strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" opacity="0.95" />;
        })}
        {hover != null && (
          <g>
            <line x1={xAt(hover)} x2={xAt(hover)} y1={padT} y2={H - padB} stroke="var(--cg-border-strong)" strokeWidth="1" />
            {series.map((s) => (
              <circle key={s.id} cx={xAt(hover)} cy={yAt(s.data[hover])} r="3" fill="var(--cg-surface)" stroke={s.color} strokeWidth="2" />
            ))}
          </g>
        )}
      </svg>
      {hover != null && (
        <div style={{ display: "flex", flexWrap: "wrap", gap: 12, marginTop: 8, fontSize: "var(--text-xs)", fontVariantNumeric: "tabular-nums" }}>
          {series.map((s) => (
            <span key={s.id} style={{ color: "var(--cg-text-muted)" }}>
              <span style={{ color: s.color, fontWeight: 600 }}>{s.name}</span> {s.data[hover]}%
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

window.FleetChart = FleetChart;
