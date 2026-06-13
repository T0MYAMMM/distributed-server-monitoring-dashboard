"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  addChannel,
  deleteChannel,
  getChannels,
  testChannel,
  updateChannel,
} from "@/lib/api/client";

const key = ["notification-channels"] as const;

// useChannels lists the configured notification channels (admin only).
export function useChannels(enabled = true) {
  return useQuery({ queryKey: key, queryFn: getChannels, enabled });
}

export function useAddChannel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: addChannel,
    onSuccess: () => qc.invalidateQueries({ queryKey: key }),
  });
}

export function useUpdateChannel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      ...body
    }: {
      id: number;
      name: string;
      config: Record<string, string>;
      enabled: boolean;
    }) => updateChannel(id, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: key }),
  });
}

export function useDeleteChannel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => deleteChannel(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: key }),
  });
}

export function useTestChannel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => testChannel(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: key }),
  });
}
