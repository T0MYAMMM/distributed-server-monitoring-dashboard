// LogsView — the "investigate" surface: server/level/module/grep controls, a
// volume histogram, and the monospace log pane with token highlighting.
function LogMessage({ line }) {
  if (line.level === "DEBUG" || line.level === "ERROR") {
    return <span className="cg-log-msg">{line.msg}</span>;
  }
  const codeClass = line.code < 300 ? "cg-tok-2xx" : line.code < 400 ? "cg-tok-3xx" : line.code < 500 ? "cg-tok-4xx" : "cg-tok-5xx";
  return (
    <span className="cg-log-msg">
      <span className="cg-tok-verb">{line.verb}</span>{" "}
      <span className="cg-tok-url">{line.url}</span>{" "}
      <span className={codeClass}>{line.code}</span>{" "}
      <span className="cg-tok-key">dur</span>=<span className="cg-tok-val">{line.dur}ms</span>
    </span>
  );
}

function Histogram({ data }) {
  const max = Math.max(...data.map((d) => d.info + d.warn + d.error));
  return (
    <div style={{ display: "flex", alignItems: "flex-end", gap: 3, height: 56 }}>
      {data.map((d, i) => {
        const total = d.info + d.warn + d.error;
        const h = (total / max) * 100;
        return (
          <div key={i} title={`${total} lines`} style={{ flex: 1, height: `${h}%`, minHeight: 2, display: "flex", flexDirection: "column", justifyContent: "flex-end", borderRadius: 2, overflow: "hidden", background: "var(--cg-muted)" }}>
            <div style={{ height: `${(d.error / total) * 100}%`, background: "var(--cg-critical)" }} />
            <div style={{ height: `${(d.warn / total) * 100}%`, background: "var(--cg-warning)" }} />
            <div style={{ height: `${(d.info / total) * 100}%`, background: "var(--cg-success)" }} />
          </div>
        );
      })}
    </div>
  );
}

function LogsView({ initialServer }) {
  const { Button, Input, Card } = window.CloudGuardDesignSystem_a66aa9;
  const Icon = window.CGIcon;
  const PageHeader = window.CGPageHeader;
  const { servers, modules, logs, histogram } = window.CGData;
  const running = servers.filter((s) => s.status === "running");
  const [serverId, setServerId] = React.useState(initialServer?.id || running[0]?.id);
  const [level, setLevel] = React.useState("");
  const [selMods, setSelMods] = React.useState([]);
  const [openMods, setOpenMods] = React.useState(false);
  const [q, setQ] = React.useState("");
  const [live, setLive] = React.useState(false);

  let rows = logs;
  if (level) rows = rows.filter((l) => l.level === level);
  if (selMods.length) rows = rows.filter((l) => selMods.includes(l.module));
  if (q) rows = rows.filter((l) => l.msg.toLowerCase().includes(q.toLowerCase()));

  const sel = { height: 40, borderRadius: "var(--radius-md)", border: "1px solid var(--cg-border)", background: "var(--cg-bg)", color: "var(--cg-text)", padding: "0 10px", fontSize: "var(--text-sm)", fontFamily: "var(--font-sans)" };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
      <PageHeader title="Logs & Activity" description="Tail, grep, and investigate per-node logs across the fleet." />
      <Card>
        <div style={{ padding: 16, display: "flex", flexDirection: "column", gap: 14 }}>
          <div style={{ display: "flex", flexWrap: "wrap", gap: 8, alignItems: "center" }}>
            <select style={sel} value={serverId} onChange={(e) => { setServerId(e.target.value); setSelMods([]); }}>
              {running.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </select>
            <select style={sel} value={level} onChange={(e) => setLevel(e.target.value)}>
              <option value="">All levels</option>
              {["DEBUG", "INFO", "WARN", "ERROR"].map((l) => <option key={l} value={l}>{l}</option>)}
            </select>
            <div style={{ position: "relative" }}>
              <button onClick={() => setOpenMods((v) => !v)} style={{ ...sel, display: "inline-flex", alignItems: "center", gap: 6, cursor: "pointer" }}>
                {selMods.length === 0 ? "All apps" : `${selMods.length} app${selMods.length > 1 ? "s" : ""}`}
                <Icon name="chevronDown" size={14} />
              </button>
              {openMods && (
                <div style={{ position: "absolute", top: 44, left: 0, zIndex: 10, width: 220, background: "var(--cg-surface-2)", border: "1px solid var(--cg-border)", borderRadius: "var(--radius-md)", boxShadow: "var(--shadow-md)", padding: 6 }}>
                  {modules.map((m) => {
                    const on = selMods.includes(m);
                    return (
                      <button key={m} onClick={() => setSelMods((c) => on ? c.filter((x) => x !== m) : [...c, m])}
                        style={{ display: "flex", alignItems: "center", gap: 8, width: "100%", textAlign: "left", border: "none", background: "transparent", color: "var(--cg-text)", padding: "7px 8px", borderRadius: "var(--radius-sm)", cursor: "pointer", fontSize: "var(--text-sm)", fontFamily: "var(--font-mono)" }}>
                        <span style={{ width: 15, height: 15, borderRadius: 4, border: "1px solid var(--cg-border-strong)", background: on ? "var(--cg-accent)" : "transparent", color: "#fff", display: "flex", alignItems: "center", justifyContent: "center" }}>{on && <Icon name="check" size={11} strokeWidth={3} />}</span>
                        {m}
                      </button>
                    );
                  })}
                </div>
              )}
            </div>
            <div style={{ position: "relative", flex: 1, minWidth: 180 }}>
              <span style={{ position: "absolute", left: 10, top: "50%", transform: "translateY(-50%)", color: "var(--cg-text-muted)", display: "flex" }}><Icon name="search" size={16} /></span>
              <Input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Grep keywords in message…" style={{ paddingLeft: 32 }} />
            </div>
            <Button variant={live ? "primary" : "outline"} size="sm" onClick={() => setLive((v) => !v)} style={{ height: 40 }}>
              <Icon name="radio" size={16} className={live ? "cg-pulse" : ""} /> {live ? "Live" : "Tail"}
            </Button>
          </div>

          <div>
            <div style={{ display: "flex", justifyContent: "space-between", fontSize: "var(--text-2xs)", color: "var(--cg-text-faint)", marginBottom: 6, textTransform: "uppercase", letterSpacing: "var(--tracking-wide)" }}>
              <span>Volume by level · last 24h</span><span>{rows.length} lines</span>
            </div>
            <Histogram data={histogram} />
          </div>

          <div className="cg-logpane" style={{ maxHeight: 360 }}>
            {rows.slice(0, 40).map((l) => (
              <div className="cg-logline" key={l.id}>
                <span className="cg-log-ts">{l.ts}</span>
                <span className="cg-log-level" data-level={l.level}>{l.level}</span>
                <span className="cg-log-module">{l.module}</span>
                <LogMessage line={l} />
              </div>
            ))}
          </div>
        </div>
      </Card>
    </div>
  );
}
window.CGLogsView = LogsView;
