"use client";

import { useQuery } from "@tanstack/react-query";
import { ShieldQuestion } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { getUnknownAgents } from "@/lib/api/client";

// UnknownAgentsCard surfaces recently rejected (unregistered) agent reports so a
// misnamed agent can be diagnosed without a packet capture (backend B2).
export function UnknownAgentsCard() {
  const { data } = useQuery({
    queryKey: ["unknown-agents"],
    queryFn: getUnknownAgents,
    refetchInterval: 15_000,
  });
  const agents = data ?? [];

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <ShieldQuestion className="h-4 w-4 text-warning" />
          Recent Unknown Agents
        </CardTitle>
      </CardHeader>
      <CardContent>
        {agents.length === 0 ? (
          <p className="text-sm text-muted-foreground">No rejected reports. All agents are reporting under registered names.</p>
        ) : (
          <ul className="space-y-2">
            {agents.map((a) => (
              <li key={a.name} className="flex items-center justify-between gap-2 text-sm">
                <div className="min-w-0">
                  <span className="font-medium">{a.name}</span>
                  <span className="ml-2 text-xs text-muted-foreground">
                    {a.remote_addr} · last seen {a.last_seen}
                  </span>
                </div>
                <Badge variant="warning">{a.count}×</Badge>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
