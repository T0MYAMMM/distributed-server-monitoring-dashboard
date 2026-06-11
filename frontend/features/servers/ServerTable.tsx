"use client";

import { Fragment, useState } from "react";
import { ChevronDown, ChevronRight, ArrowDownToLine, ArrowUpFromLine } from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { ResourceBar } from "@/components/shared/ResourceBar";
import { cn } from "@/lib/utils";
import { countryToFlag, formatBytesPerSec, formatPercent, formatUptime } from "@/lib/format";
import type { Server } from "@/lib/api/types";

function DetailItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="truncate font-medium">{value || "—"}</dd>
    </div>
  );
}

export function ServerTable({
  servers,
  loading,
  limit,
}: {
  servers: Server[] | undefined;
  loading?: boolean;
  limit?: number;
}) {
  const [expanded, setExpanded] = useState<string | null>(null);
  const rows = limit ? (servers ?? []).slice(0, limit) : servers ?? [];

  if (loading && !servers) {
    return (
      <div className="space-y-2 p-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-8" />
          <TableHead>Status</TableHead>
          <TableHead>Server</TableHead>
          <TableHead className="hidden sm:table-cell">Location</TableHead>
          <TableHead className="hidden min-w-[120px] md:table-cell">CPU</TableHead>
          <TableHead className="hidden min-w-[120px] md:table-cell">Memory</TableHead>
          <TableHead className="hidden min-w-[120px] lg:table-cell">Disk</TableHead>
          <TableHead className="hidden lg:table-cell">Network</TableHead>
          <TableHead className="hidden sm:table-cell">Uptime</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((s) => {
          const open = expanded === s.id;
          return (
            <Fragment key={s.id}>
              <TableRow
                className="cursor-pointer"
                onClick={() => setExpanded(open ? null : s.id)}
              >
                <TableCell className="text-muted-foreground">
                  {open ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                </TableCell>
                <TableCell>
                  <StatusBadge status={s.status} />
                </TableCell>
                <TableCell>
                  <div className="font-medium">{s.name}</div>
                  <div className="text-xs text-muted-foreground">{s.os_type}</div>
                </TableCell>
                <TableCell className="hidden sm:table-cell">
                  <span className="mr-1">{countryToFlag(s.location)}</span>
                  <span className="text-muted-foreground">{s.location}</span>
                </TableCell>
                <TableCell className="hidden md:table-cell">
                  <ResourceBar value={s.cpu} />
                </TableCell>
                <TableCell className="hidden md:table-cell">
                  <ResourceBar value={s.memory} />
                </TableCell>
                <TableCell className="hidden lg:table-cell">
                  <ResourceBar value={s.disk} />
                </TableCell>
                <TableCell className="hidden whitespace-nowrap text-xs text-muted-foreground lg:table-cell">
                  <span className="flex items-center gap-1">
                    <ArrowDownToLine className="h-3 w-3" />
                    {formatBytesPerSec(s.network_in)}
                  </span>
                  <span className="flex items-center gap-1">
                    <ArrowUpFromLine className="h-3 w-3" />
                    {formatBytesPerSec(s.network_out)}
                  </span>
                </TableCell>
                <TableCell className="hidden whitespace-nowrap text-sm tabular-nums sm:table-cell">
                  {formatUptime(s.uptime)}
                </TableCell>
              </TableRow>
              {open && (
                <TableRow className="bg-muted/30 hover:bg-muted/30">
                  <TableCell colSpan={9}>
                    <dl className="grid grid-cols-2 gap-4 py-2 text-sm sm:grid-cols-3 lg:grid-cols-4">
                      <DetailItem label="Hostname" value={s.hostname} />
                      <DetailItem label="OS" value={s.os_type} />
                      <DetailItem label="CPU" value={s.cpu_info} />
                      <DetailItem label="Type" value={s.type} />
                      <DetailItem label="Total memory" value={`${s.total_memory} GB`} />
                      <DetailItem label="Total disk" value={`${s.total_disk} GB`} />
                      <DetailItem label="Tailscale IP" value={s.tailscale_ip} />
                      <DetailItem label="Public IP" value={s.ip_address} />
                      <DetailItem label="Disk usage" value={formatPercent(s.disk)} />
                    </dl>
                  </TableCell>
                </TableRow>
              )}
            </Fragment>
          );
        })}
      </TableBody>
    </Table>
  );
}
