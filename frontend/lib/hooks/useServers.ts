"use client";

import { useEffect, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { getServers } from "@/lib/api/client";
import { DashboardSocket } from "@/lib/ws/manager";
import { onAuthChange } from "@/lib/auth";

export const serversKey = ["servers"] as const;

// useServers returns the live fleet. TanStack Query owns the data; a
// WebSocket patches the cache on push, with 5s REST polling as the fallback
// when the socket is unavailable.
export function useServers() {
  const qc = useQueryClient();
  const [wsConnected, setWsConnected] = useState(false);

  const query = useQuery({
    queryKey: serversKey,
    queryFn: getServers,
    // Polling backs up the socket; harmless duplication when the socket is live.
    refetchInterval: 5_000,
  });

  useEffect(() => {
    let socket: DashboardSocket | null = null;
    const connect = () => {
      socket?.close();
      socket = new DashboardSocket({
        onServers: (servers) => qc.setQueryData(serversKey, servers),
        onConnectionChange: setWsConnected,
      });
      socket.connect();
    };
    connect();
    // Re-handshake when auth changes so IP masking stays correct live.
    const unsub = onAuthChange(connect);
    return () => {
      unsub();
      socket?.close();
    };
  }, [qc]);

  return { ...query, wsConnected };
}
