"use client";

import { useMemo, useState } from "react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { Activity, Download, ScrollText, TrendingUp } from "lucide-react";
import { PageHeader } from "@/components/shared/PageHeader";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { ResourceBar } from "@/components/shared/ResourceBar";
import { EmptyState } from "@/components/shared/EmptyState";
import { useServerStats, useLogVolume, useTopModules } from "@/lib/hooks/useAnalytics";
import type { AnalyticsRange, ServerStat } from "@/lib/api/types";

const RANGES: AnalyticsRange[] = ["24h", "7d", "30d", "90d"];

const LEVELS = [
  { key: "error", label: "Error", color: "hsl(var(--critical))" },
  { key: "warn", label: "Warn", color: "hsl(var(--warning))" },
  { key: "info", label: "Info", color: "hsl(var(--success))" },
  { key: "debug", label: "Debug", color: "hsl(var(--muted-foreground))" },
] as const;

export default function AnalyticsPage() {
  const [range, setRange] = useState<AnalyticsRange>("7d");
  const { data: servers, isLoading: serversLoading } = useServerStats(range);
  const { data: volume, isError: volumeOff } = useLogVolume(range);
  const { data: topModules } = useTopModules(range);

  const kpis = useMemo(() => {
    const list = servers ?? [];
    const running = list.filter((s) => s.status === "running").length;
    const avgUptime = list.length
      ? list.reduce((a, s) => a + s.uptime_percent, 0) / list.length
      : 0;
    const totalLines = (volume ?? []).reduce((a, p) => a + p.debug + p.info + p.warn + p.error, 0);
    const errors = (volume ?? []).reduce((a, p) => a + p.error, 0);
    const errorRate = totalLines ? (errors / totalLines) * 100 : 0;
    return { running, total: list.length, avgUptime, totalLines, errorRate };
  }, [servers, volume]);

  const chartData = useMemo(
    () =>
      (volume ?? []).map((p) => ({
        ...p,
        label: formatBucket(p.ts, range),
      })),
    [volume, range],
  );

  const exportCsv = () => {
    const rows = [
      ["server", "status", "uptime_percent", "cpu", "memory", "disk", "disk_days_to_full"],
      ...(servers ?? []).map((s) => [
        s.name,
        s.status,
        s.uptime_percent.toFixed(1),
        s.cpu.toFixed(1),
        s.memory.toFixed(1),
        s.disk.toFixed(1),
        s.disk_days_to_full < 0 ? "" : s.disk_days_to_full.toFixed(0),
      ]),
    ];
    downloadCsv(`cloudguard-analytics-${range}.csv`, rows);
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Analytics"
        description="Trends and capacity across the fleet, beyond the live dashboard."
        actions={
          <div className="flex items-center gap-2">
            <div className="flex gap-1">
              {RANGES.map((r) => (
                <Button
                  key={r}
                  size="sm"
                  variant={r === range ? "secondary" : "ghost"}
                  onClick={() => setRange(r)}
                >
                  {r}
                </Button>
              ))}
            </div>
            <Button size="sm" variant="outline" onClick={exportCsv}>
              <Download className="h-4 w-4" /> CSV
            </Button>
          </div>
        }
      />

      {/* KPIs */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Kpi icon={Activity} label="Servers reporting" value={`${kpis.running}/${kpis.total}`} />
        <Kpi icon={TrendingUp} label="Avg uptime" value={`${kpis.avgUptime.toFixed(1)}%`} />
        <Kpi icon={ScrollText} label="Log lines" value={volumeOff ? "—" : compact(kpis.totalLines)} />
        <Kpi icon={ScrollText} label="Error rate" value={volumeOff ? "—" : `${kpis.errorRate.toFixed(1)}%`} />
      </div>

      {/* Log volume */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Log volume by level</CardTitle>
        </CardHeader>
        <CardContent>
          {volumeOff ? (
            <EmptyState
              Icon={ScrollText}
              title="Logs are not enabled"
              description="Set LOG_DATABASE_URL on the hub to collect logs and unlock log analytics."
            />
          ) : chartData.length === 0 ? (
            <EmptyState Icon={ScrollText} title="No log data in this range" description="Try a wider range, or ship logs from a node via Integrations." />
          ) : (
            <>
              <div className="mb-3 flex flex-wrap gap-4">
                {LEVELS.map((l) => (
                  <span key={l.key} className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <span className="h-2 w-2 rounded-full" style={{ background: l.color }} />
                    {l.label}
                  </span>
                ))}
              </div>
              <ResponsiveContainer width="100%" height={256}>
                <AreaChart data={chartData} margin={{ top: 4, right: 8, bottom: 0, left: -16 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" vertical={false} />
                  <XAxis dataKey="label" stroke="hsl(var(--muted-foreground))" fontSize={11} tickLine={false} minTickGap={24} />
                  <YAxis stroke="hsl(var(--muted-foreground))" fontSize={11} tickLine={false} />
                  <Tooltip
                    contentStyle={{
                      background: "hsl(var(--popover))",
                      border: "1px solid hsl(var(--border))",
                      borderRadius: 8,
                      fontSize: 12,
                    }}
                  />
                  {LEVELS.map((l) => (
                    <Area
                      key={l.key}
                      type="monotone"
                      dataKey={l.key}
                      name={l.label}
                      stackId="1"
                      stroke={l.color}
                      fill={l.color}
                      fillOpacity={0.25}
                      isAnimationActive={false}
                    />
                  ))}
                </AreaChart>
              </ResponsiveContainer>
            </>
          )}
        </CardContent>
      </Card>

      <div className="grid gap-6 lg:grid-cols-5">
        {/* Per-server table */}
        <Card className="lg:col-span-3">
          <CardHeader>
            <CardTitle className="text-base">Servers — uptime &amp; capacity</CardTitle>
          </CardHeader>
          <CardContent className="px-0">
            {serversLoading && !servers ? (
              <div className="space-y-2 px-6">
                {Array.from({ length: 5 }).map((_, i) => (
                  <Skeleton key={i} className="h-10 w-full" />
                ))}
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border text-left text-xs text-muted-foreground">
                      <th className="px-6 py-2 font-medium">Server</th>
                      <th className="px-3 py-2 font-medium">Uptime</th>
                      <th className="px-3 py-2 font-medium">Disk</th>
                      <th className="px-6 py-2 font-medium">Capacity</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(servers ?? []).map((s) => (
                      <tr key={s.id} className="border-b border-border/50 last:border-0">
                        <td className="px-6 py-2.5">
                          <div className="flex items-center gap-2">
                            <StatusBadge status={s.status} />
                            <span className="font-medium">{s.name}</span>
                          </div>
                        </td>
                        <td className="px-3 py-2.5 tabular-nums">{s.uptime_percent.toFixed(1)}%</td>
                        <td className="w-32 px-3 py-2.5">
                          <ResourceBar value={s.disk} />
                        </td>
                        <td className="px-6 py-2.5">
                          <Capacity stat={s} />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Top error sources */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle className="text-base">Top error sources</CardTitle>
          </CardHeader>
          <CardContent>
            {volumeOff ? (
              <p className="text-sm text-muted-foreground">Logs are not enabled.</p>
            ) : (topModules ?? []).length === 0 ? (
              <p className="text-sm text-muted-foreground">No module activity in this range.</p>
            ) : (
              <div className="space-y-2.5">
                {topModules!.map((m) => {
                  const rate = m.total ? (m.errors / m.total) * 100 : 0;
                  return (
                    <div key={m.module} className="flex items-center justify-between gap-3 text-sm">
                      <span className="truncate font-mono text-xs">{m.module}</span>
                      <span className="flex shrink-0 items-center gap-2">
                        <span className="text-muted-foreground">{compact(m.total)}</span>
                        {m.errors > 0 && (
                          <Badge variant={rate > 5 ? "critical" : "warning"}>{m.errors} err</Badge>
                        )}
                      </span>
                    </div>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function Kpi({
  icon: Icon,
  label,
  value,
}: {
  icon: typeof Activity;
  label: string;
  value: string;
}) {
  return (
    <Card>
      <CardContent className="flex items-center gap-3 p-4">
        <span className="flex h-10 w-10 items-center justify-center rounded-md bg-primary/10 text-primary">
          <Icon className="h-5 w-5" />
        </span>
        <div>
          <p className="text-xs text-muted-foreground">{label}</p>
          <p className="text-xl font-semibold tabular-nums">{value}</p>
        </div>
      </CardContent>
    </Card>
  );
}

function Capacity({ stat }: { stat: ServerStat }) {
  if (stat.disk_days_to_full < 0) {
    return <span className="text-sm text-muted-foreground">Stable</span>;
  }
  const days = Math.round(stat.disk_days_to_full);
  if (days <= 14) {
    return <Badge variant="critical">~{days}d to full</Badge>;
  }
  if (days <= 60) {
    return <Badge variant="warning">~{days}d to full</Badge>;
  }
  return <span className="text-sm text-muted-foreground">~{days}d to full</span>;
}

function formatBucket(ts: number, range: AnalyticsRange): string {
  const d = new Date(ts * 1000);
  if (range === "24h") return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  return d.toLocaleDateString([], { month: "short", day: "numeric" });
}

function compact(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`;
  return String(n);
}

function downloadCsv(filename: string, rows: (string | number)[][]) {
  const csv = rows
    .map((r) => r.map((c) => `"${String(c).replace(/"/g, '""')}"`).join(","))
    .join("\n");
  const blob = new Blob([csv], { type: "text/csv" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
