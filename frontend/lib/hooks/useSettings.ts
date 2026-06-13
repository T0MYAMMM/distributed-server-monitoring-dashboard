"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getSettings, updateSettings } from "@/lib/api/client";

// useSettings loads the editable settings doc + About facts.
export function useSettings() {
  return useQuery({ queryKey: ["settings"], queryFn: getSettings });
}

// useUpdateSettings persists overrides and refreshes the cached doc with the
// server's authoritative result (effective values after env precedence).
export function useUpdateSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: Record<string, string>) => updateSettings(values),
    onSuccess: (doc) => qc.setQueryData(["settings"], doc),
  });
}
