# Redesign & Rebranding Plan

A product, brand, and design plan to take this from "a working self-hosted
monitor" to **a product**. It also specifies the **unfinished features** as
design briefs.

> **How to use this doc (for design agents).** Sections 4 and 5 are a series of
> **Design Briefs**, each self-contained with a consistent shape:
> *Goal · Users · Data & API · Layout direction · States · Interactions ·
> Acceptance criteria · Dependencies*. Pick a brief, design the interface/flow,
> and produce the component(s). Honor the design-system constraints in §4.1 and
> the brand in §3. Build constraints worth remembering: Next.js App Router +
> Tailwind + shadcn/Radix; **all color via design tokens, never hex**; dark-first;
> TanStack Query owns server state with a WebSocket patching the cache;
> accessibility is non-negotiable (keyboard nav, focus rings, color is never the
> only status signal).

---

## 1. Vision & positioning

**One-liner.** A private, two-minute-to-onboard observability cockpit for a
small fleet — **metrics, alerts, and logs in one pane**, riding entirely on a
Tailscale tailnet, with zero runtime dependencies on the hub.

**Why it wins (the wedge).** Prometheus+Grafana+Loki is heavyweight; SaaS sends
your data off-network and bills per host; per-box tools (htop/glances) don't
aggregate. This product is the "open one page, see every machine — and now *read
its logs* — installed in two minutes, never leaves your network" option.

**North-star experience.** From "I have N boxes doing things" to "I can see all
of them, know the moment one misbehaves, and read why — without leaving my
tailnet or standing up a stack."

**Primary persona — "the solo operator / small team."** Runs 3–30 machines
(VPSes, home servers, scrapers, side-projects). Comfortable in a terminal, wants
a calm dashboard, hates yak-shaving an observability stack. Secondary: a
teammate/viewer who should see health but not secrets.

---

## 2. Current state (what we're redesigning around)

**Shipped & solid.** AppShell (collapsible sidebar, top bar, command palette,
theme toggle); **Dashboard** (KPI cards + trends, fleet time-series, resource
panel, server table); **Servers** (table + per-server chart, drag-and-drop
ordering); **Alerts** (list, severity filter, acknowledge, live bell badge);
**Admin** (clients, copy-paste install commands, unknown-agents, password reset);
**Logs & Activity** (per-node, **multi-select module**, level filter, **keyword
grep**, **live SSE tail**); **Login** (first-run init). Backend: layered Go,
metrics history + rollups, alerts + webhook notifier, external-Postgres logs,
`/api/v1`, IP masking, auth-aware WebSocket.

**Placeholders (nav items that route to "coming soon").** Analytics,
Integrations, Feedback, Help, Settings.

**Known gaps / debt to design against.**
- Single shared admin credential — **no users/roles/audit**.
- No **per-server detail** view (only an expandable row); no mobile card layout
  rebuilt on the new system.
- Alert behavior is backend-config only — **no rules UI**; one webhook, no
  channel management UI.
- Logs are powerful but raw — **no saved views, no insights/patterns, no
  retention controls in-app**.
- Onboarding is copy-paste commands — **no guided "add a node" wizard**.
- Command palette is minimal; data-viz theming is basic; density is fixed.

---

## 3. Rebranding

### 3.1 Brand positioning
- **Category:** private fleet observability (metrics + alerts + logs).
- **Promise:** *See everything, privately, in two minutes.*
- **Pillars:** Private by default · Effortless onboarding · Calm clarity ·
  Zero-dependency.
- **Personality / voice:** calm, technical, trustworthy, plain-spoken. No
  growth-hack hype, no fear-based alerting language. Confident and quiet.

### 3.2 Naming
"CloudGuard" is a **working placeholder** — generic, crowded, trademark-risky,
and "Guard"/"Cloud" undersell the private-tailnet angle. Treat naming as an open
decision; below are **criteria + directions**, not a mandate.

**Naming criteria.** Short (≤2 syllables ideal), ownable (domain/npm/social
plausibly free), evokes *private + fleet + calm visibility*, not another
"…Watch/…Guard/…Metrics", easy to say, works as a verb-ish UI word.

**Directions + candidates (for the team to vet for availability/trademark):**
- *Tailnet/private-fleet:* **Tailwatch**, **Privet**, **Enclave**, **Homestead**.
- *Calm/clarity/observability:* **Lumen**, **Vantage**, **Clearwell**, **Beacon**,
  **Tide**.
- *Fleet/constellation:* **Fleetline**, **Polestar**, **Cohort**, **Muster**.
- *Keep & own:* refine **CloudGuard** with a distinct mark + tagline if legal is
  comfortable.

**Recommendation.** Shortlist 3 (e.g. **Vantage**, **Enclave**, **Beacon**),
check trademark/domain, then lock one. The codebase uses "CloudGuard" in exactly
one brand surface (the sidebar wordmark, `app/icon.svg`, the `<title>`) plus
copy — renaming is a small, centralized change.

### 3.3 Visual identity direction
- **Logo.** The current **Activity-pulse mark** (a heartbeat line in a rounded
  orange tile) is a strong, ownable seed — evolve it into a refined wordmark +
  glyph. The pulse = "your fleet's vitals," which is on-strategy. Deliver:
  app icon, wordmark (light/dark), monochrome, favicon (favicon already ships
  from `app/icon.svg`).
- **Color.** Keep **dark-first** with **one accent**. The orange accent is warm
  and distinctive; either own it or test a calmer signature (deep teal /
  electric indigo) against the "calm" personality. Lock a **semantic palette**:
  success=green, warning=amber, critical=red, plus a **data-viz series palette**
  (5–6 distinguishable, colorblind-safe hues) for charts.
- **Typography.** A crisp grotesk for UI (e.g. Geist/Inter) + a **monospace for
  logs/metrics/IPs** (e.g. Geist Mono/JetBrains Mono). One type scale, 8px
  spacing rhythm (already in place).
- **Motion.** Subtle, purposeful (state changes, live-tail, the pulse mark).
  Reduced-motion respected.
- **Imagery/illustration.** Minimal; favor real data + tasteful empty states
  over stock illustration.

### 3.4 Brand applications (deliverables)
Login screen, AppShell wordmark, empty states, the favicon (done), a one-page
landing/README hero, and a consistent **voice guide** for microcopy (buttons,
toasts, empty/error states).

---

## 4. Redesign — design system & screens

### 4.1 Design-system evolution
- **Tokens.** Formalize the existing CSS-variable tokens into a documented scale:
  surfaces/elevation, semantic colors, the data-viz series palette, radii,
  spacing, and a **density toggle** (comfortable/compact) for data-dense tables.
- **Mono surface.** Define a logs/metrics monospace surface style (line height,
  selection, wrap vs truncate, level-badge colors, token highlighting for
  HTTP verbs/status/URLs/`[key=value]` tags à la log-geulis).
- **Components to add/upgrade** (shadcn/Radix): Sheet/Drawer, Tabs, Tooltip,
  Combobox, **DateRange picker**, Pagination, richer **DataTable** (column
  visibility, sticky header, row selection, CSV export), **ChartPanel** theming
  (legend, hover tooltip, brush/zoom), Sparkline, **StatusTimeline**, Empty/Error
  states, ConfirmDialog, Stepper/Wizard. (Button, Card, Badge, Dialog,
  DropdownMenu+Checkbox, Table, Skeleton, Toast, Command exist.)
- **Patterns.** Loading skeletons everywhere, explicit empty/error states,
  optimistic mutations with toasts, URL-synced filters (shareable views),
  keyboard shortcuts surfaced in the command palette.

### 4.2 Screen briefs

#### Brief — AppShell & navigation
- **Goal.** A calm, dense-friendly shell that scales as features land.
- **Users.** Everyone. **Data/API.** `config/nav.ts`, unacked alert count.
- **Layout.** Keep sidebar (groups, collapse, account card) + top bar (search →
  ⌘K, bell badge, theme, language). Add: per-node/global context, a "fleet
  status" pill in the bar (N running / M down), and a place for org/workspace
  once multi-user lands.
- **States.** Collapsed rail, mobile off-canvas, active highlight, "coming soon"
  treatment that previews value (not a dead end).
- **Acceptance.** Adding a feature = one nav entry + one route; keyboard
  navigable; badge reflects live alerts.

#### Brief — Dashboard (overview)
- **Goal.** Answer "is my fleet OK?" in 3 seconds, then "what's interesting?"
- **Data/API.** `useServers`, `useFleetSummary(range)`, `useAlerts`.
- **Layout.** Personalized header + global uptime KPI; 4 KPI cards with trend
  badges; **fleet time-series** (per-series legend, range selector 1h/6h/24h/7d,
  chart-type toggle, rich hover); **resource + anomaly** panel that links to the
  offending server; full-width **server summary** table (top N + "view all").
- **Upgrades.** True **multi-series fleet chart** (one line per server/group, not
  a single representative); customizable card row; "what changed" anomaly feed.
- **States.** Skeletons, empty fleet (onboarding CTA), partial data.
- **Acceptance.** Trends/uptime driven by real history; anomaly cards deep-link;
  responsive 4→2→1.

#### Brief — Servers (fleet) + **Server detail (NEW)**
- **Goal.** Manage and inspect the fleet; drill into one machine.
- **Data/API.** `useServers`, `useServerMetrics(id,range)`, status/order/delete.
- **Fleet.** TanStack table: region/status filters, sort, **CSV export**, column
  visibility, row checkboxes (future bulk actions), inline mini bars/sparklines,
  uptime %, **drag-and-drop ordering** (shipped). Cards on mobile.
- **Server detail (drawer or `/servers/[id]`) — net-new.** Header (status, OS,
  location, Tailscale/public IP per auth), its own metric charts with range,
  recent **alerts for this host**, recent **logs for this host** (link into Logs
  filtered), admin actions (force status, delete-with-confirm), install/agent
  info. This is the highest-value missing surface.
- **Acceptance.** Detail reachable from row, command palette, and anomaly links;
  masking respected; mobile cards preserve expand behavior.

#### Brief — Alerts & Incidents
- **Goal.** Triage fast; configure what alerts mean.
- **Data/API.** `useAlerts`, acknowledge; **(new)** alert-rules + channels CRUD.
- **Layout.** Severity filter + search; grouped/incident view (collapse repeats
  into an incident with a count + timeline); acknowledge / assign / snooze;
  per-alert detail with the triggering server + a StatusTimeline.
- **Net-new (see §5).** An **Alert Rules** builder and **Notification Channels**
  manager — alerts shouldn't be backend-env-only.
- **Acceptance.** Live updates; bell badge consistent; ack is optimistic;
  color+icon+text for severity.

#### Brief — Logs & Activity (enrich)
- **Goal.** From "tail and grep" to "investigate."
- **Data/API.** `getServerLogs` (level/module[]/q/since/until), `/logs/modules`,
  SSE `/logs/stream`; see `docs/logs.md`.
- **Shipped.** Per-node, **multi-select module**, level filter, **message-only
  keyword grep**, **live tail**, monospace pane, level badges.
- **Upgrades.** **Time-range picker** + histogram of volume by level (click to
  zoom); **saved views** (server+modules+query as a shareable URL); token
  highlighting (HTTP verb/status/URL/`[key=value]`); expandable rows for
  multi-line/tracebacks; per-line copy/permalink; "logs for this server" entry
  from Server detail; module groups (e.g. all `tlusr-*`, all `claude-agents-*`).
- **Acceptance.** Filters URL-synced/shareable; tail respects all filters
  (already true server-side); virtualized for 10k+ lines.

#### Brief — Login & first-run
- **Goal.** A confident first impression and a 2-minute path to first data.
- **Layout.** Branded split screen; first-run "set admin password" → then a
  **guided onboarding wizard** (see §5) instead of dropping into an empty admin.
- **Acceptance.** Distinguishes init vs login; errors are kind; leads to value.

#### Brief — Command palette (⌘K)
- **Goal.** Power-user navigation + actions.
- **Upgrades.** Recent/most-used; actions (add server, ack all, toggle theme,
  jump to a node's logs/detail); fuzzy across servers, modules, alerts.

#### Brief — Mobile
- **Goal.** Health at a glance on a phone.
- **Layout.** Sidebar → off-canvas; KPI 4→1; tables → cards (rebuild
  `MobileServerCard` on the new primitives); logs pane full-height with sticky
  controls.

---

## 5. Unfinished features — design briefs

Each is a nav placeholder today (or a strategic gap). Build order in §6.

#### Brief — Settings (replaces the `/settings` placeholder)
- **Goal.** One home for configuration currently only reachable via env/CLI.
- **Sections.** *General* (instance name, theme/density defaults, time zone);
  *Data & retention* (raw/rollup windows, **log retention**, log DB status);
  *Thresholds* (default disk/cpu alert thresholds, stale-after seconds);
  *Security* (admin password, session length, **mask Tailscale IPs** toggle);
  *About* (version, build, health).
- **Data/API.** New `GET/PUT /api/v1/settings` (back the currently-env-only
  values; keep env as override). **States/Acceptance.** Each setting has a
  sensible default, inline validation, optimistic save + toast, and a "restart
  required" hint where applicable.

#### Brief — Integrations (`/integrations`)
- **Goal.** Connect outbound channels and inbound sources.
- **Catalog cards.** Slack, Discord, ntfy, Email/SMTP, Generic Webhook,
  PagerDuty (outbound); plus "log sources" presets (journald, Docker, file/glob)
  that **generate the agent `--logs` snippet / bridge** for a node (productize the
  cookbook in `docs/logs.md`).
- **Data/API.** `notification_channels` CRUD; test-send; per-channel enable.
- **Acceptance.** Add/test/enable a channel in <1 min; secrets stored server-side;
  status (last delivery, failures) shown.

#### Brief — Notification Channels + **Alert Rules** (powers Alerts & Integrations)
- **Goal.** Make alerting user-configurable (today: env webhook + fixed rules).
- **Alert Rules builder.** Conditions on metric/status (e.g. `disk > 90% for
  5m`, `status → stopped`, `cpu > 80% for 10m`), scope (all / group / server),
  severity, and **which channels** fire. Live preview of matching servers.
- **Data/API.** `alert_rules` + `notification_channels` CRUD; evaluate rules in
  the alerts service (extends the shipped foundation).
- **Acceptance.** Create a rule → it appears in Alerts when it fires → routes to
  chosen channels; rules are testable/dry-runnable.

#### Brief — Analytics (`/analytics`)
- **Goal.** Trends and reports beyond the live dashboard.
- **Content.** Fleet utilization over 7/30 days, uptime/SLO per server, busiest
  hours, **log volume by module/level over time**, top error sources, capacity
  hints ("db-1 disk trending to full in ~9 days"). Exportable.
- **Data/API.** Extend metrics summary + rollups; add log-aggregate queries
  (counts by module/level/time over the log DB).
- **Acceptance.** Date-range driven; charts themed via tokens; export CSV/PNG.

#### Brief — Users & Roles (RBAC) + audit  *(strategic — removes the biggest gap)*
- **Goal.** Replace the single shared password with real accounts.
- **Roles.** Owner / Admin (mutate) / Viewer (read, IP-masked). Invite flow,
  per-user sessions, **audit log** of admin actions.
- **Data/API.** `users`, `sessions`, `audit_log`; extend auth service + JWT
  claims (subject/role); gate mutations by role (the masking + v1-auth seams
  already exist).
- **Acceptance.** Viewer never sees real IPs (REST + WS); admin actions audited;
  invite/reset flows.

#### Brief — Onboarding wizard ("Add a node")
- **Goal.** Turn copy-paste commands into a guided flow.
- **Steps.** Name the node → pick OS/arch → copy the exact install command (or
  one-liner) → live "waiting for first report…" → success → optional **"ship
  logs"** step that emits the right `--logs`/bridge snippet (recipe picker from
  `docs/logs.md`).
- **Acceptance.** From click to live node without leaving the screen; detects
  arrival via polling/WS.

#### Brief — Help (`/help`) & Feedback (`/feedback`)
- **Help.** In-app docs/runbook surfacing (architecture, add-machine, logs
  cookbook, keyboard shortcuts), searchable; contextual "?" links per page.
- **Feedback.** Lightweight in-app feedback (category + message → webhook/issue),
  plus a changelog/"what's new".
- **Acceptance.** No dead-end "coming soon"; Help is genuinely useful;
  Feedback submits somewhere real.

#### Brief — Public/shareable status page  *(stretch)*
- **Goal.** A read-only, optionally-public health page for a subset of servers.
- **Acceptance.** Per-server opt-in; no secrets; its own minimal theme.

---

## 6. Roadmap & phasing

- **Phase A — Brand & system.** Lock name + identity; formalize tokens/density
  + data-viz palette; mono log surface; component upgrades (Sheet, DateRange,
  DataTable, ChartPanel). Polish AppShell, Dashboard, Login.
- **Phase B — Finish the core.** **Server detail** view; **Logs insights**
  (time range + histogram + saved views + token highlighting); **Alert Rules +
  Notification Channels** + Alerts incident view; mobile cards.
- **Phase C — Configurability.** **Settings** + `GET/PUT /api/v1/settings`;
  **Integrations** catalog; **Onboarding wizard**; Help/Feedback.
- **Phase D — Team & insight.** **RBAC/users/audit**; **Analytics**; status page.

Sequence rationale: brand/system unblocks every screen; Server detail + Logs
insights deliver the most user value on top of shipped backend; configurability
removes env/CLI-only friction; RBAC/analytics open team use.

---

## 7. Work packets for design agents

| # | Brief | Type | Priority | Depends on | Done when |
|---|---|---|---|---|---|
| A1 | Brand identity (name, logo, color, type, voice) | brand | P0 | — | tokens + assets + voice guide delivered |
| A2 | Design system v2 (tokens, density, data-viz, mono, components) | system | P0 | A1 | documented kit + coded primitives |
| A3 | AppShell + Dashboard polish | screen | P1 | A2 | redesigned, responsive, token-only |
| B1 | **Server detail** drawer/page | screen | P1 | A2 | reachable, charts+alerts+logs per host |
| B2 | **Logs insights** (range, histogram, saved views, highlighting) | screen | P1 | A2 | filters URL-synced, virtualized |
| B3 | **Alert Rules + Notification Channels** | feature | P1 | A2 | create→fire→route, testable |
| B4 | Alerts incident view + Mobile cards | screen | P2 | A2 | grouping + responsive |
| C1 | **Settings** + `/api/v1/settings` | feature | P2 | A2 | env values editable in-app | ✅ shipped |
| C2 | **Integrations** catalog (+ log-source snippet gen) | feature | P2 | B3 | add/test/enable channel | ✅ shipped |
| C3 | **Onboarding wizard** | flow | P2 | A2 | click→live node→ship logs | |
| C4 | Help + Feedback | screen | P3 | A2 | useful, submits real | ✅ shipped |
| D1 | **RBAC / users / audit** | feature | P2 | A2 | roles enforced, viewer masked, audited | |
| D2 | Analytics | screen | P3 | A2 | reports + export | ✅ shipped |
| D3 | Public status page | screen | P3 | D1 | opt-in, no secrets | |

Each packet should ship: the interface design (per §4.1 constraints + §3 brand),
the states (loading/empty/error), and—where a brief names an API—the request/
response shape so backend and frontend can proceed in parallel.

> **Status (June 2026).** The five "coming soon" surfaces are now built and live:
> **Settings** (C1), **Integrations** channels + log-source generator (C2),
> **Help** + **Feedback** (C4), and **Analytics** (D2). C2 ships the notification
> **channels** half of B3 (CRUD + per-type delivery + test-send, fanned out from
> the alerts service); the full **alert-rules builder** in B3 remains open, as do
> the onboarding wizard (C3), RBAC (D1), and the public status page (D3).
