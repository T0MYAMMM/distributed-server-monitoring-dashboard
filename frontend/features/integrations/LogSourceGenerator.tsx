"use client";

import { useMemo, useState } from "react";
import { Check, Copy, FileText } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { API_URL } from "@/config/config";
import { useServers } from "@/lib/hooks/useServers";
import { cn } from "@/lib/utils";

type SourceKind = "file" | "journald" | "docker" | "glob";

const KINDS: { kind: SourceKind; label: string; blurb: string }[] = [
  { kind: "file", label: "Log file", blurb: "The app already writes a log file." },
  { kind: "journald", label: "journald", blurb: "A systemd service logging to the journal." },
  { kind: "docker", label: "Docker", blurb: "Containers using the json-file driver." },
  { kind: "glob", label: "Rotating files", blurb: "Files that appear dynamically (a glob)." },
];

const LIVE = "/var/log/cloudguard/node.log";

interface Block {
  label: string;
  lang: string;
  code: string;
}

function buildBlocks(kind: SourceKind, node: string, hub: string, input: string): Block[] {
  const name = node || "<node>";
  const agent = "/opt/server-monitor-agent/monitor-agent";
  const exec = (logs: string) =>
    `# /etc/systemd/system/server-monitor-agent.service.d/override.conf\n[Service]\nExecStart=\nExecStart=${agent} --name ${name} \\\n  --server ${hub} --interval 2s --logs ${logs}`;

  if (kind === "file") {
    const paths = input.trim() || "/var/log/app/app.log";
    return [{ label: "Agent unit override — then daemon-reload + restart the agent", lang: "ini", code: exec(paths) }];
  }

  if (kind === "journald") {
    const units = (input.trim() || "app-api app-web")
      .split(/[\s,]+/)
      .filter(Boolean)
      .map((u) => `-u ${u}`)
      .join(" ");
    const bridge = `# /usr/local/bin/cloudguard-logbridge.sh  (run as a system service; uses the unit name as the module)
#!/usr/bin/env bash
set -euo pipefail
mkdir -p "$(dirname ${LIVE})"
trap 'kill 0' EXIT
journalctl -n 0 -f -o json --no-pager ${units} \\
  | python3 -u -c '
import sys, json, datetime, re
ANSI = re.compile(r"\\x1b\\[[0-9;]*[A-Za-z]")
for line in sys.stdin:
    try: e = json.loads(line)
    except Exception: continue
    unit = (e.get("_SYSTEMD_UNIT","") or e.get("SYSLOG_IDENTIFIER","")).removesuffix(".service")
    if unit in ("init.scope",""): continue
    msg = e.get("MESSAGE","")
    if isinstance(msg, list): msg = bytes(msg).decode("utf-8","replace")
    msg = ANSI.sub("", msg)
    ts = datetime.datetime.fromtimestamp(int(e["__REALTIME_TIMESTAMP"])/1e6, datetime.timezone.utc).isoformat()
    print(f"{ts} | INFO | {unit} | {msg}", flush=True)
' >> ${LIVE}`;
    return [
      { label: "1. Bridge script — journald → log-geulis file", lang: "bash", code: bridge },
      { label: "2. Point the agent at the bridge output", lang: "ini", code: exec(LIVE) },
    ];
  }

  if (kind === "docker") {
    const filter = input.trim();
    const psFilter = filter ? ` --filter "name=${filter}"` : "";
    const bridge = `# /usr/local/bin/cloudguard-docker-logbridge.sh  (container name as the module)
#!/usr/bin/env bash
set -euo pipefail
mkdir -p "$(dirname ${LIVE})"
trap 'kill 0' EXIT
declare -A PIDS
strip() { python3 -u -c '
import sys, re, datetime
ANSI = re.compile(r"\\x1b\\[[0-9;]*[A-Za-z]")
name = sys.argv[1]
for line in sys.stdin:
    msg = ANSI.sub("", line.rstrip("\\n"))
    ts = datetime.datetime.now(datetime.timezone.utc).isoformat()
    print(f"{ts} | INFO | {name} | {msg}", flush=True)
' "$1"; }
while true; do
  for c in $(docker ps${psFilter} --format '{{.Names}}'); do
    [[ -n "\${PIDS[$c]:-}" ]] && kill -0 "\${PIDS[$c]}" 2>/dev/null && continue
    ( docker logs -f --since 3s "$c" 2>&1 | strip "$c" >> ${LIVE} ) &
    PIDS[$c]=$!
  done
  sleep 5
done`;
    return [
      { label: "1. Docker bridge — reconciles docker ps every 5s", lang: "bash", code: bridge },
      { label: "2. Point the agent at the bridge output", lang: "ini", code: exec(LIVE) },
    ];
  }

  // glob
  const pattern = input.trim() || "/tmp/job_*.log";
  const bridge = `# /usr/local/bin/cloudguard-glob-logbridge.sh  (filename as the module)
#!/usr/bin/env bash
set -euo pipefail
mkdir -p "$(dirname ${LIVE})"
trap 'kill 0' EXIT
declare -A SEEN
while true; do
  for f in ${pattern}; do
    [[ -e "$f" ]] || continue
    [[ -n "\${SEEN[$f]:-}" ]] && continue
    SEEN[$f]=1
    camp="$(basename "$f" .log)"
    ( tail -F -n 0 "$f" | sed -u "s/^/$(date -u +%FT%TZ) | INFO | $camp | /" >> ${LIVE} ) &
  done
  sleep 5
done`;
  return [
    { label: "1. Glob bridge — follows each matching file", lang: "bash", code: bridge },
    { label: "2. Point the agent at the bridge output", lang: "ini", code: exec(LIVE) },
  ];
}

// LogSourceGenerator turns a node + source type into the exact agent --logs
// snippet (and bridge, when needed), productizing the docs/logs.md cookbook.
export function LogSourceGenerator() {
  const { data: servers } = useServers();
  const [kind, setKind] = useState<SourceKind>("file");
  const [node, setNode] = useState("");
  const [input, setInput] = useState("");

  const hub = API_URL || "http://<hub-ip>:5000";
  const placeholder: Record<SourceKind, string> = {
    file: "/var/log/nginx/access.log,/var/log/app/app.log",
    journald: "app-api app-web",
    docker: "(optional) container name filter",
    glob: "/tmp/job_*.log",
  };
  const inputLabel: Record<SourceKind, string> = {
    file: "Log file path(s), comma-separated",
    journald: "systemd unit(s)",
    docker: "Container name filter",
    glob: "Glob pattern",
  };

  const blocks = useMemo(() => buildBlocks(kind, node, hub, input), [kind, node, hub, input]);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <FileText className="h-4 w-4 text-primary" /> Log source setup
        </CardTitle>
        <CardDescription>
          Generate the agent <code className="font-mono text-xs">--logs</code> snippet (and bridge) to stream a
          node&apos;s logs into Logs &amp; Activity. Register the node first under Admin.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="grid gap-2 sm:grid-cols-4">
          {KINDS.map((k) => (
            <button
              key={k.kind}
              onClick={() => setKind(k.kind)}
              className={cn(
                "rounded-md border p-3 text-left transition-colors",
                kind === k.kind ? "border-primary bg-primary/10" : "border-border hover:bg-accent",
              )}
            >
              <p className="text-sm font-medium">{k.label}</p>
              <p className="mt-0.5 text-xs text-muted-foreground">{k.blurb}</p>
            </button>
          ))}
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label htmlFor="gen-node">Node (agent --name)</Label>
            <select
              id="gen-node"
              value={node}
              onChange={(e) => setNode(e.target.value)}
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            >
              <option value="">Select a node…</option>
              {(servers ?? []).map((s) => (
                <option key={s.id} value={s.name}>
                  {s.name}
                </option>
              ))}
            </select>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="gen-input">{inputLabel[kind]}</Label>
            <Input
              id="gen-input"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={placeholder[kind]}
            />
          </div>
        </div>

        <div className="space-y-4">
          {blocks.map((b) => (
            <CodeBlock key={b.label} block={b} />
          ))}
        </div>
        <p className="text-xs text-muted-foreground">
          After editing the agent unit: <code className="font-mono">sudo systemctl daemon-reload &amp;&amp; sudo
          systemctl restart server-monitor-agent</code>. Bridges should run as their own service so they survive
          reboots — see docs/logs.md §3.
        </p>
      </CardContent>
    </Card>
  );
}

function CodeBlock({ block }: { block: Block }) {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard.writeText(block.code).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    });
  };
  return (
    <div>
      <div className="mb-1.5 flex items-center justify-between">
        <p className="text-xs font-medium text-muted-foreground">{block.label}</p>
        <Button size="sm" variant="ghost" className="h-7 px-2 text-xs" onClick={copy}>
          {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
          {copied ? "Copied" : "Copy"}
        </Button>
      </div>
      <pre className="cg-logpane overflow-x-auto rounded-md border border-border bg-muted/40 p-3 text-xs leading-relaxed">
        <code>{block.code}</code>
      </pre>
    </div>
  );
}
