"use client";

import Link from "next/link";
import {
  Activity,
  AlertTriangle,
  Bell,
  Cpu,
  MemoryStick,
  Network,
  Plus,
  RefreshCw,
  Server as ServerIcon,
} from "lucide-react";
import { PageHeader } from "@/components/shared/PageHeader";
import { StatCard } from "@/components/shared/StatCard";
import { ResourceBar } from "@/components/shared/ResourceBar";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ServerTable } from "@/features/servers/ServerTable";
import { MetricsChart } from "@/features/metrics/MetricsChart";
import { useServers } from "@/lib/hooks/useServers";
import { useFleetSummary } from "@/lib/hooks/useMetrics";
import { formatBytesPerSec, formatPercent } from "@/lib/format";

export default function DashboardPage() {
  const { data: servers, isLoading: serversLoading, refetch } = useServers();
  const { data: summary, isLoading: summaryLoading } = useFleetSummary("24h");

  const list = servers ?? [];
  const focusServer = list.find((s) => s.status === "running") ?? list[0];
  const stopped = list.filter((s) => s.status === "stopped");
  const highDisk = list.filter((s) => s.disk >= 90);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Welcome back, Admin"
        description={
          <span>
            Global uptime{" "}
            <span className="font-medium text-success">{formatPercent(summary?.uptime_percent ?? 0, 1)}</span>{" "}
            over the last 24h
          </span>
        }
        actions={
          <>
            <Button asChild>
              <Link href="/admin">
                <Plus className="h-4 w-4" /> Add Server
              </Link>
            </Button>
            <Button variant="outline" asChild>
              <Link href="/alerts">
                <Bell className="h-4 w-4" /> View Alerts
              </Link>
            </Button>
            <Button variant="ghost" size="icon" aria-label="Refresh" onClick={() => refetch()}>
              <RefreshCw className="h-4 w-4" />
            </Button>
          </>
        }
      />

      {/* KPI cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          label="Active Servers"
          Icon={ServerIcon}
          loading={summaryLoading}
          value={`${summary?.active_servers ?? 0}/${summary?.total_servers ?? 0}`}
        />
        <StatCard
          label="Avg CPU Load"
          Icon={Cpu}
          loading={summaryLoading}
          value={formatPercent(summary?.cpu.value ?? 0)}
          delta={summary?.cpu.delta}
          deltaSuffix="%"
          goodDirection="down"
        />
        <StatCard
          label="Avg Memory Usage"
          Icon={MemoryStick}
          loading={summaryLoading}
          value={formatPercent(summary?.memory.value ?? 0)}
          delta={summary?.memory.delta}
          deltaSuffix="%"
          goodDirection="down"
        />
        <StatCard
          label="Aggregate Network"
          Icon={Network}
          loading={summaryLoading}
          value={formatBytesPerSec(summary?.network.value ?? 0)}
        />
      </div>

      {/* Main grid: chart + resource usage */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <MetricsChart serverId={focusServer?.id} title={focusServer ? `Resource trend — ${focusServer.name}` : "Resource trend"} />
        </div>
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Resource Usage</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <ResourceBar label="CPU" value={summary?.cpu.value ?? 0} />
            <ResourceBar label="Memory" value={summary?.memory.value ?? 0} />
            <ResourceBar label="Disk I/O" value={summary?.disk.value ?? 0} />
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Network</span>
              <span className="font-medium">{formatBytesPerSec(summary?.network.value ?? 0)}</span>
            </div>

            <div className="rounded-md border border-border bg-muted/30 p-3">
              <p className="mb-2 flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
                <Activity className="h-3.5 w-3.5" /> Notices
              </p>
              {stopped.length === 0 && highDisk.length === 0 ? (
                <p className="text-sm text-muted-foreground">All systems nominal.</p>
              ) : (
                <ul className="space-y-1.5 text-sm">
                  {stopped.length > 0 && (
                    <li>
                      <Link href="/servers" className="flex items-center gap-1.5 text-critical hover:underline">
                        <AlertTriangle className="h-3.5 w-3.5" />
                        {stopped.length} server{stopped.length > 1 ? "s" : ""} stopped
                      </Link>
                    </li>
                  )}
                  {highDisk.map((s) => (
                    <li key={s.id}>
                      <Link href="/servers" className="flex items-center gap-1.5 text-warning hover:underline">
                        <AlertTriangle className="h-3.5 w-3.5" />
                        Disk &gt; 90% on {s.name}
                      </Link>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Server summary table */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <CardTitle className="text-base">Servers</CardTitle>
          <Button variant="ghost" size="sm" asChild>
            <Link href="/servers">View all</Link>
          </Button>
        </CardHeader>
        <CardContent className="px-0 pb-0">
          <ServerTable servers={servers} loading={serversLoading} limit={6} />
        </CardContent>
      </Card>
    </div>
  );
}
