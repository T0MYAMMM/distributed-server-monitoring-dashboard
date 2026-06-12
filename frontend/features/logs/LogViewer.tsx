"use client";

import { useEffect, useRef, useState } from "react";
import { Radio, ScrollText, Search } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/shared/EmptyState";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { ApiError, getServerLogs, logsStreamUrl } from "@/lib/api/client";
import { useServers } from "@/lib/hooks/useServers";
import type { LogLine } from "@/lib/api/types";

const LEVELS = ["", "DEBUG", "INFO", "WARN", "ERROR"] as const;

function levelClass(level: string): string {
  switch (level.toUpperCase()) {
    case "ERROR":
      return "text-critical";
    case "WARN":
      return "text-warning";
    case "DEBUG":
      return "text-muted-foreground";
    default:
      return "text-success";
  }
}

export function LogViewer() {
  const { data: servers } = useServers();
  const [serverId, setServerId] = useState("");
  const [level, setLevel] = useState("");
  const [q, setQ] = useState("");
  const [live, setLive] = useState(false);
  const [lines, setLines] = useState<LogLine[]>([]);
  const [disabled, setDisabled] = useState(false);
  const [loading, setLoading] = useState(false);
  const lastId = useRef(0);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!serverId && servers && servers.length > 0) setServerId(servers[0].id);
  }, [servers, serverId]);

  useEffect(() => {
    if (!serverId) return;
    let cancelled = false;
    let es: EventSource | null = null;
    let interval: ReturnType<typeof setInterval> | undefined;
    setLines([]);
    setDisabled(false);

    const loadOnce = async () => {
      setLoading(true);
      try {
        const data = await getServerLogs(serverId, { level, q, limit: 500 });
        if (cancelled) return;
        const chrono = [...data].reverse(); // oldest first for a terminal feel
        lastId.current = chrono.length ? chrono[chrono.length - 1].id : 0;
        setLines(chrono);
      } catch (e) {
        if (e instanceof ApiError && e.status === 503) setDisabled(true);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    (async () => {
      await loadOnce();
      if (cancelled || disabled) return;
      if (live) {
        es = new EventSource(logsStreamUrl(serverId, lastId.current));
        es.onmessage = (ev) => {
          try {
            const l = JSON.parse(ev.data) as LogLine;
            lastId.current = l.id;
            setLines((cur) => [...cur, l].slice(-1000));
          } catch {
            /* ignore */
          }
        };
      } else {
        interval = setInterval(loadOnce, 5000);
      }
    })();

    return () => {
      cancelled = true;
      es?.close();
      if (interval) clearInterval(interval);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [serverId, level, q, live]);

  useEffect(() => {
    if (live && scrollRef.current) scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
  }, [lines, live]);

  return (
    <Card>
      <CardContent className="space-y-3 p-4">
        {/* Controls */}
        <div className="flex flex-wrap items-center gap-2">
          <select
            value={serverId}
            onChange={(e) => setServerId(e.target.value)}
            className="h-9 rounded-md border border-input bg-background px-2 text-sm"
            aria-label="Server"
          >
            {(servers ?? []).map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </select>
          <select
            value={level}
            onChange={(e) => setLevel(e.target.value)}
            className="h-9 rounded-md border border-input bg-background px-2 text-sm"
            aria-label="Level"
          >
            {LEVELS.map((l) => (
              <option key={l} value={l}>
                {l || "All levels"}
              </option>
            ))}
          </select>
          <div className="relative min-w-[180px] flex-1">
            <Search className="absolute left-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={q}
              onChange={(e) => setQ(e.target.value)}
              placeholder="Search message or module…"
              className="h-9 pl-8"
            />
          </div>
          <Button
            variant={live ? "default" : "outline"}
            size="sm"
            onClick={() => setLive((v) => !v)}
          >
            <Radio className={cn("h-4 w-4", live && "animate-pulse")} />
            {live ? "Live" : "Tail"}
          </Button>
        </div>

        {/* Output */}
        {disabled ? (
          <EmptyState
            Icon={ScrollText}
            title="Logs are not enabled"
            description="Set LOG_DATABASE_URL on the hub (e.g. the home-db Postgres) and run agents with --logs to ship log files here."
          />
        ) : loading && lines.length === 0 ? (
          <div className="space-y-1.5">
            {Array.from({ length: 8 }).map((_, i) => (
              <Skeleton key={i} className="h-5 w-full" />
            ))}
          </div>
        ) : lines.length === 0 ? (
          <EmptyState Icon={ScrollText} title="No logs yet" description="No matching log lines for this server." />
        ) : (
          <div
            ref={scrollRef}
            className="max-h-[60vh] overflow-auto rounded-md border border-border bg-background/50 p-2 font-mono text-xs leading-relaxed"
          >
            {lines.map((l) => (
              <div key={l.id} className="flex gap-2 whitespace-pre-wrap break-words px-1 py-0.5 hover:bg-muted/40">
                <span className="shrink-0 text-muted-foreground">{l.ts.replace("T", " ").replace("Z", "")}</span>
                <Badge variant="outline" className={cn("shrink-0 px-1.5 py-0 font-semibold", levelClass(l.level))}>
                  {l.level}
                </Badge>
                <span className="shrink-0 text-primary">{l.module}</span>
                <span className="text-foreground">{l.message}</span>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
