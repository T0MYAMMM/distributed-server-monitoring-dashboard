"use client";

import { useState } from "react";
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/shared/EmptyState";
import { useServerMetrics } from "@/lib/hooks/useMetrics";
import { LineChart as LineChartIcon } from "lucide-react";
import type { MetricRange } from "@/lib/api/types";

const RANGES: MetricRange[] = ["1h", "6h", "24h", "7d"];

// Colors resolve through the design tokens (the browser resolves the CSS vars in
// the SVG stroke), so the chart never hardcodes hex values.
const SERIES = [
  { key: "cpu", label: "CPU", color: "var(--cg-series-1)" },
  { key: "memory", label: "Memory", color: "var(--cg-series-2)" },
  { key: "disk", label: "Disk", color: "var(--cg-series-3)" },
] as const;

function formatTick(ts: number, range: MetricRange): string {
  const d = new Date(ts * 1000);
  if (range === "7d") return d.toLocaleDateString([], { month: "short", day: "numeric" });
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

export function MetricsChart({ serverId, title }: { serverId?: string; title?: string }) {
  const [range, setRange] = useState<MetricRange>("24h");
  const { data, isLoading } = useServerMetrics(serverId, range);
  const points = (data ?? []).map((d) => ({ ...d }));

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-2 space-y-0">
        <CardTitle className="text-base">{title ?? "Resource trend"}</CardTitle>
        <div className="flex gap-1">
          {RANGES.map((r) => (
            <Button
              key={r}
              size="sm"
              variant={r === range ? "secondary" : "ghost"}
              className="h-7 px-2 text-xs"
              onClick={() => setRange(r)}
            >
              {r}
            </Button>
          ))}
        </div>
      </CardHeader>
      <CardContent>
        {!serverId ? (
          <EmptyState Icon={LineChartIcon} title="No server selected" description="Pick a server to see its trend." />
        ) : isLoading && points.length === 0 ? (
          <Skeleton className="h-64 w-full" />
        ) : points.length === 0 ? (
          <EmptyState Icon={LineChartIcon} title="No history yet" description="Metrics appear here once the agent has reported for a while." />
        ) : (
          <>
            <div className="mb-3 flex flex-wrap gap-4">
              {SERIES.map((s) => (
                <span key={s.key} className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <span className="h-2 w-2 rounded-full" style={{ background: s.color }} />
                  {s.label}
                </span>
              ))}
            </div>
            <ResponsiveContainer width="100%" height={256}>
              <LineChart data={points} margin={{ top: 4, right: 8, bottom: 0, left: -16 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" vertical={false} />
                <XAxis
                  dataKey="ts"
                  tickFormatter={(v) => formatTick(v, range)}
                  stroke="hsl(var(--muted-foreground))"
                  fontSize={11}
                  tickLine={false}
                />
                <YAxis domain={[0, 100]} stroke="hsl(var(--muted-foreground))" fontSize={11} tickLine={false} />
                <Tooltip
                  contentStyle={{
                    background: "hsl(var(--popover))",
                    border: "1px solid hsl(var(--border))",
                    borderRadius: 8,
                    fontSize: 12,
                  }}
                  labelFormatter={(v) => formatTick(Number(v), range)}
                  formatter={(value: number, name: string) => [`${Number(value).toFixed(1)}%`, name]}
                />
                {SERIES.map((s) => (
                  <Line
                    key={s.key}
                    type="monotone"
                    dataKey={s.key}
                    name={s.label}
                    stroke={s.color}
                    strokeWidth={2}
                    dot={false}
                    isAnimationActive={false}
                  />
                ))}
              </LineChart>
            </ResponsiveContainer>
          </>
        )}
      </CardContent>
    </Card>
  );
}
