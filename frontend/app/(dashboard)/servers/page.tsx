"use client";

import { RefreshCw } from "lucide-react";
import { PageHeader } from "@/components/shared/PageHeader";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { ServerTable } from "@/features/servers/ServerTable";
import { MetricsChart } from "@/features/metrics/MetricsChart";
import { useServers } from "@/lib/hooks/useServers";

export default function ServersPage() {
  const { data, isLoading, refetch } = useServers();
  const list = data ?? [];
  const focus = list.find((s) => s.status === "running") ?? list[0];

  return (
    <div className="space-y-6">
      <PageHeader
        title="Servers"
        description={`${list.length} machine${list.length === 1 ? "" : "s"} in the fleet`}
        actions={
          <Button variant="ghost" size="icon" aria-label="Refresh" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4" />
          </Button>
        }
      />
      {focus && <MetricsChart serverId={focus.id} title={`Resource trend — ${focus.name}`} />}
      <Card>
        <CardContent className="px-0 pb-0">
          <ServerTable servers={data} loading={isLoading} />
        </CardContent>
      </Card>
    </div>
  );
}
