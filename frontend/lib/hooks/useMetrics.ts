"use client";

import { useQuery } from "@tanstack/react-query";
import { getFleetSummary, getServerMetrics } from "@/lib/api/client";
import type { MetricRange } from "@/lib/api/types";

// useFleetSummary drives the dashboard KPI cards and trend badges.
export function useFleetSummary(range: MetricRange = "24h") {
  return useQuery({
    queryKey: ["fleet-summary", range],
    queryFn: () => getFleetSummary(range),
    refetchInterval: 15_000,
  });
}

// useServerMetrics drives the per-server time-series chart.
export function useServerMetrics(id: string | undefined, range: MetricRange) {
  return useQuery({
    queryKey: ["server-metrics", id, range],
    queryFn: () => getServerMetrics(id as string, range),
    enabled: Boolean(id),
    refetchInterval: 15_000,
  });
}
