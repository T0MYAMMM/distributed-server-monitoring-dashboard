"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { acknowledgeAlert, getAlerts } from "@/lib/api/client";

// useAlerts drives the alerts page and the topbar bell badge.
export function useAlerts(severity?: string) {
  return useQuery({
    queryKey: ["alerts", severity ?? "all"],
    queryFn: () => getAlerts(severity),
    refetchInterval: 10_000,
  });
}

// useAcknowledgeAlert acknowledges an alert and refreshes the list.
export function useAcknowledgeAlert() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => acknowledgeAlert(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alerts"] }),
  });
}

// useUnacknowledgedCount feeds the topbar bell badge.
export function useUnacknowledgedCount(): number {
  const { data } = useAlerts();
  return (data ?? []).filter((a) => a.acknowledged_at === "").length;
}
