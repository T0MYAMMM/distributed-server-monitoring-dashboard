"use client";

import { useQuery } from "@tanstack/react-query";
import { getAnalyticsServers, getLogVolume, getTopModules } from "@/lib/api/client";
import type { AnalyticsRange } from "@/lib/api/types";

// useServerStats drives the per-server uptime + capacity table.
export function useServerStats(range: AnalyticsRange) {
  return useQuery({
    queryKey: ["analytics", "servers", range],
    queryFn: () => getAnalyticsServers(range),
    refetchInterval: 30_000,
  });
}

// useLogVolume drives the volume-by-level histogram. Disabled-logs (503) surfaces
// as an error the page renders as an empty/disabled state.
export function useLogVolume(range: AnalyticsRange, server?: string) {
  return useQuery({
    queryKey: ["analytics", "log-volume", range, server ?? "all"],
    queryFn: () => getLogVolume(range, server),
    retry: false,
  });
}

// useTopModules drives the "top error sources" list.
export function useTopModules(range: AnalyticsRange, server?: string) {
  return useQuery({
    queryKey: ["analytics", "top-modules", range, server ?? "all"],
    queryFn: () => getTopModules(range, server),
    retry: false,
  });
}
