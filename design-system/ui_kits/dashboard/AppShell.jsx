// AppShell — sidebar (brand, nav groups, account card) + topbar (search → ⌘K,
// fleet-status pill, bell badge, theme, language). Mirrors the shipped layout.
const NAV_MAIN = [
  { key: "dashboard", title: "Dashboard", icon: "dashboard" },
  { key: "servers", title: "Servers", icon: "server" },
  { key: "alerts", title: "Alerts & Incidents", icon: "bell" },
  { key: "logs", title: "Logs & Activity", icon: "logs" },
  { key: "analytics", title: "Analytics", icon: "analytics", soon: true },
  { key: "integrations", title: "Integrations", icon: "plug", soon: true },
];
const NAV_SUPPORT = [
  { key: "feedback", title: "Feedback", icon: "feedback", soon: true },
  { key: "help", title: "Help", icon: "help", soon: true },
  { key: "settings", title: "Settings", icon: "settings", soon: true },
  { key: "admin", title: "Admin", icon: "shield" },
];

function NavLink({ item, active, collapsed, onClick }) {
  const Icon = window.CGIcon;
  return (
    <button onClick={() => onClick(item.key)} title={item.title}
      className="cg-navlink"
      data-active={active ? "" : undefined}
      style={{
        display: "flex", alignItems: "center", gap: 12, width: "100%",
        padding: collapsed ? "8px" : "8px 12px", justifyContent: collapsed ? "center" : "flex-start",
        borderRadius: "var(--radius-md)", border: "none", cursor: "pointer",
        fontFamily: "var(--font-sans)", fontSize: "var(--text-sm)", fontWeight: 500, textAlign: "left",
        background: active ? "var(--cg-accent)" : "transparent",
        color: active ? "var(--cg-accent-fg)" : "var(--cg-text-muted)",
      }}>
      <Icon name={item.icon} size={18} style={{ flexShrink: 0 }} />
      {!collapsed && <span style={{ flex: 1, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>{item.title}</span>}
      {!collapsed && item.soon && (
        <span style={{ fontSize: "var(--text-2xs)", padding: "1px 6px", borderRadius: 4, background: "var(--cg-muted)", color: "var(--cg-text-muted)" }}>Soon</span>
      )}
    </button>
  );
}

function Sidebar({ active, onNavigate, collapsed, onToggle }) {
  const Icon = window.CGIcon;
  return (
    <aside style={{ width: collapsed ? 64 : 248, flexShrink: 0, background: "var(--cg-surface)", borderRight: "1px solid var(--cg-border)", display: "flex", flexDirection: "column", height: "100%", transition: "width .18s" }}>
      <div style={{ height: 64, display: "flex", alignItems: "center", gap: 8, padding: collapsed ? "0" : "0 16px", justifyContent: collapsed ? "center" : "flex-start", borderBottom: "1px solid var(--cg-border)" }}>
        <span style={{ width: 32, height: 32, borderRadius: 8, background: "var(--cg-accent)", color: "#fff", display: "flex", alignItems: "center", justifyContent: "center", flexShrink: 0 }}>
          <Icon name="activity" size={20} strokeWidth={2.4} />
        </span>
        {!collapsed && <span style={{ fontSize: "var(--text-base)", fontWeight: 600, letterSpacing: "var(--tracking-tight)", color: "var(--cg-text)" }}>CloudGuard</span>}
        {!collapsed && (
          <button onClick={onToggle} aria-label="Collapse sidebar" style={{ marginLeft: "auto", background: "none", border: "none", cursor: "pointer", color: "var(--cg-text-muted)", display: "flex", padding: 4 }}>
            <Icon name="panelLeft" size={16} />
          </button>
        )}
      </div>
      <nav style={{ flex: 1, overflowY: "auto", padding: "16px 12px", display: "flex", flexDirection: "column", gap: 24 }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
          {!collapsed && <p style={{ margin: "0 0 4px", padding: "0 12px", fontSize: "var(--text-2xs)", fontWeight: 500, textTransform: "uppercase", letterSpacing: "var(--tracking-wide)", color: "var(--cg-text-faint)" }}>Main Navigation</p>}
          {NAV_MAIN.map((it) => <NavLink key={it.key} item={it} active={active === it.key} collapsed={collapsed} onClick={onNavigate} />)}
        </div>
        <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
          {!collapsed && <p style={{ margin: "0 0 4px", padding: "0 12px", fontSize: "var(--text-2xs)", fontWeight: 500, textTransform: "uppercase", letterSpacing: "var(--tracking-wide)", color: "var(--cg-text-faint)" }}>Support</p>}
          {NAV_SUPPORT.map((it) => <NavLink key={it.key} item={it} active={active === it.key} collapsed={collapsed} onClick={onNavigate} />)}
        </div>
      </nav>
      <div style={{ borderTop: "1px solid var(--cg-border)", padding: 12 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 12, padding: "8px", borderRadius: "var(--radius-md)", justifyContent: collapsed ? "center" : "flex-start" }}>
          <Icon name="user" size={28} style={{ color: "var(--cg-text-muted)", flexShrink: 0 }} />
          {!collapsed && (
            <div style={{ minWidth: 0 }}>
              <p style={{ margin: 0, fontSize: "var(--text-sm)", fontWeight: 500, color: "var(--cg-text)" }}>Admin</p>
              <p style={{ margin: 0, fontSize: "var(--text-xs)", color: "var(--cg-text-muted)" }}>Manage fleet</p>
            </div>
          )}
        </div>
      </div>
    </aside>
  );
}

function Topbar({ running, total, unacked, onSearch }) {
  const Icon = window.CGIcon;
  return (
    <header style={{ height: 64, flexShrink: 0, display: "flex", alignItems: "center", gap: 12, padding: "0 16px", borderBottom: "1px solid var(--cg-border)", background: "color-mix(in srgb, var(--cg-bg) 80%, transparent)", backdropFilter: "blur(8px)", position: "sticky", top: 0, zIndex: 5 }}>
      <button onClick={onSearch} style={{ display: "flex", alignItems: "center", gap: 8, height: 36, maxWidth: 420, flex: 1, padding: "0 12px", borderRadius: "var(--radius-md)", border: "1px solid var(--cg-border)", background: "var(--cg-surface)", color: "var(--cg-text-muted)", cursor: "pointer", fontFamily: "var(--font-sans)", fontSize: "var(--text-sm)" }}>
        <Icon name="search" size={16} />
        <span style={{ flex: 1, textAlign: "left" }}>Search…</span>
        <kbd style={{ fontFamily: "var(--font-mono)", fontSize: "var(--text-2xs)", border: "1px solid var(--cg-border)", borderRadius: 4, padding: "1px 5px", background: "var(--cg-muted)" }}>⌘K</kbd>
      </button>
      <div style={{ marginLeft: "auto", display: "flex", alignItems: "center", gap: 10 }}>
        <span style={{ display: "inline-flex", alignItems: "center", gap: 8, height: 32, padding: "0 12px", borderRadius: "var(--radius-pill)", border: "1px solid var(--cg-border)", background: "var(--cg-surface)", fontSize: "var(--text-xs)", color: "var(--cg-text-muted)" }}>
          <span style={{ width: 7, height: 7, borderRadius: "50%", background: "var(--cg-success)" }} />
          <span style={{ color: "var(--cg-text)", fontWeight: 500 }}>{running}</span> running
          <span style={{ color: "var(--cg-border-strong)" }}>·</span>
          <span style={{ color: "var(--cg-text)", fontWeight: 500 }}>{total - running}</span> down
        </span>
        <button style={{ position: "relative", width: 40, height: 40, display: "flex", alignItems: "center", justifyContent: "center", borderRadius: "var(--radius-md)", border: "none", background: "transparent", color: "var(--cg-text)", cursor: "pointer" }}>
          <Icon name="bell" size={20} />
          {unacked > 0 && <span style={{ position: "absolute", top: 6, right: 6, minWidth: 16, height: 16, padding: "0 4px", borderRadius: 8, background: "var(--cg-critical)", color: "#fff", fontSize: "var(--text-2xs)", fontWeight: 600, display: "flex", alignItems: "center", justifyContent: "center" }}>{unacked}</span>}
        </button>
        <button style={{ width: 40, height: 40, display: "flex", alignItems: "center", justifyContent: "center", borderRadius: "var(--radius-md)", border: "none", background: "transparent", color: "var(--cg-text)", cursor: "pointer" }}>
          <Icon name="sun" size={20} />
        </button>
        <button style={{ width: 40, height: 40, display: "flex", alignItems: "center", justifyContent: "center", borderRadius: "var(--radius-md)", border: "none", background: "transparent", color: "var(--cg-text)", cursor: "pointer" }}>
          <Icon name="globe" size={20} />
        </button>
      </div>
    </header>
  );
}

window.CGSidebar = Sidebar;
window.CGTopbar = Topbar;
