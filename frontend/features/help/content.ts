import {
  Bell,
  Network,
  ScrollText,
  ServerCog,
  SlidersHorizontal,
  type LucideIcon,
} from "lucide-react";

// One help article. Kept as plain data so the page can render and search it.
export interface HelpArticle {
  id: string;
  title: string;
  icon: LucideIcon;
  summary: string;
  paragraphs?: string[];
  steps?: string[];
  code?: string;
  link?: { label: string; href: string };
  tags: string[];
}

export const HELP_ARTICLES: HelpArticle[] = [
  {
    id: "architecture",
    title: "How CloudGuard works",
    icon: Network,
    summary: "A hub plus lightweight agents, all on your Tailscale tailnet.",
    paragraphs: [
      "Each network has one hub (this dashboard + its API) reachable on its Tailscale IP. Agents run on every monitored machine and POST metrics every couple of seconds; the hub stores them in SQLite and pushes live updates to the dashboard over WebSocket.",
      "The tailnet is the security boundary: reads are open to anyone on the tailnet, while admin actions require the Admin password. Public and Tailscale IPs are masked for anonymous viewers (toggle in Settings).",
      "Logs are optional and live in an external Postgres so the core hub stays a single zero-dependency binary.",
    ],
    tags: ["architecture", "hub", "agent", "tailscale", "websocket", "overview"],
  },
  {
    id: "add-machine",
    title: "Add a machine",
    icon: ServerCog,
    summary: "Register a node, install the agent, and watch it come online.",
    steps: [
      "In Admin, add the client by the exact name the agent will report under.",
      "On the machine, download the agent binary from the hub's /download endpoint.",
      "Run the agent with --name <node> and --server http://<hub-ip>:5000.",
      "The node appears as Pending, then flips to Running once the first report lands.",
    ],
    code: "curl -fsSL http://<hub-ip>:5000/download/monitor-agent-linux-amd64 -o monitor-agent\nchmod +x monitor-agent\n./monitor-agent --name <node> --server http://<hub-ip>:5000 --interval 2s",
    tags: ["add", "machine", "server", "agent", "register", "install", "onboard"],
  },
  {
    id: "logs",
    title: "Stream a node's logs",
    icon: ScrollText,
    summary: "Wire app, journald, Docker, or rotating-file logs into Logs & Activity.",
    paragraphs: [
      "The agent ships any files passed to --logs in the log-geulis format (TS | LEVEL | MODULE | MESSAGE). Apps that aren't plain files (journald, Docker) use a small bridge that writes a file the agent tails.",
      "Use the generator on the Integrations page to produce the exact agent snippet and bridge for your source — then run the bridge as its own service so it survives reboots.",
    ],
    link: { label: "Open the log-source generator", href: "/integrations" },
    tags: ["logs", "journald", "docker", "bridge", "tail", "cookbook", "modules"],
  },
  {
    id: "alerts",
    title: "Alerts & notification channels",
    icon: Bell,
    summary: "How incidents fire and where they're delivered.",
    paragraphs: [
      "CloudGuard raises alerts when a server goes down (stops reporting) or a threshold is breached (e.g. disk above the configured percent). They show on the Alerts page and the topbar bell.",
      "Add Slack, Discord, ntfy, a webhook, PagerDuty, or email on the Integrations page. Every enabled channel receives each alert; use Test to confirm delivery before relying on it.",
    ],
    link: { label: "Manage channels", href: "/integrations" },
    tags: ["alerts", "incidents", "threshold", "notifications", "slack", "channels"],
  },
  {
    id: "settings",
    title: "Configure the instance",
    icon: SlidersHorizontal,
    summary: "Tune thresholds, retention, masking, and branding without the CLI.",
    paragraphs: [
      "Settings surfaces values that used to be environment-only: instance name and theme, the disk alert threshold and stale-after window, log retention, and IP masking. Anything set by an environment variable stays authoritative and is shown locked.",
      "Changes that can apply live (masking, disk threshold) take effect immediately; a few are marked 'restart' where the process reads them at boot.",
    ],
    link: { label: "Open Settings", href: "/settings" },
    tags: ["settings", "configuration", "threshold", "retention", "masking", "theme"],
  },
];

export interface Shortcut {
  keys: string[];
  action: string;
}

export const SHORTCUTS: Shortcut[] = [
  { keys: ["⌘", "K"], action: "Open the command palette (Ctrl+K on Windows/Linux)" },
  { keys: ["Esc"], action: "Close the palette or an open dialog" },
  { keys: ["↑", "↓"], action: "Move between command palette results" },
  { keys: ["Enter"], action: "Run the highlighted command" },
];
