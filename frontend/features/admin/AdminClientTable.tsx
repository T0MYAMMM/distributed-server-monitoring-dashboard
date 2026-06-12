"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { Trash2 } from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { CopyButton } from "@/components/shared/CopyButton";
import { EmptyState } from "@/components/shared/EmptyState";
import { ServerOff } from "lucide-react";
import { API_URL } from "@/config/config";
import { deleteServer, setServerOrder } from "@/lib/api/client";
import { serversKey } from "@/lib/hooks/useServers";
import type { Server } from "@/lib/api/types";

function linuxCommand(name: string): string {
  return `curl -fsSL ${API_URL}/download/install_agent.sh | sudo bash -s ${name} ${API_URL}`;
}
function windowsCommand(name: string): string {
  return `iwr ${API_URL}/download/install_agent.ps1 -OutFile install_agent.ps1; .\\install_agent.ps1 -NodeName ${name} -ServerUrl ${API_URL}`;
}

export function AdminClientTable({ servers }: { servers: Server[] | undefined }) {
  const qc = useQueryClient();
  const [orders, setOrders] = useState<Record<string, string>>({});

  const invalidate = () => qc.invalidateQueries({ queryKey: serversKey });

  const del = useMutation({
    mutationFn: (id: string) => deleteServer(id),
    onSuccess: () => {
      toast.success("Server removed");
      invalidate();
    },
    onError: (e: Error) => toast.error(e.message || "Failed to delete"),
  });

  const order = useMutation({
    mutationFn: ({ id, value }: { id: string; value: number }) => setServerOrder(id, value),
    onSuccess: () => {
      toast.success("Order updated");
      invalidate();
    },
    onError: (e: Error) => toast.error(e.message || "Failed to update order"),
  });

  const rows = servers ?? [];
  if (rows.length === 0) {
    return <EmptyState Icon={ServerOff} title="No clients yet" description="Register a client to get an install command." />;
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Status</TableHead>
          <TableHead className="w-24">Order</TableHead>
          <TableHead>Install command</TableHead>
          <TableHead className="w-16 text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((s) => (
          <TableRow key={s.id}>
            <TableCell className="font-medium">{s.name}</TableCell>
            <TableCell>
              <StatusBadge status={s.status} />
            </TableCell>
            <TableCell>
              <Input
                type="number"
                className="h-8 w-20"
                value={orders[s.id] ?? String(s.order_index)}
                onChange={(e) => setOrders((o) => ({ ...o, [s.id]: e.target.value }))}
                onBlur={(e) => {
                  const v = parseInt(e.target.value, 10);
                  if (!Number.isNaN(v) && v !== s.order_index) order.mutate({ id: s.id, value: v });
                }}
              />
            </TableCell>
            <TableCell>
              <div className="flex gap-2">
                <CopyButton value={linuxCommand(s.name)} label="Linux" />
                <CopyButton value={windowsCommand(s.name)} label="Windows" />
              </div>
            </TableCell>
            <TableCell className="text-right">
              <Button
                variant="ghost"
                size="icon"
                aria-label={`Delete ${s.name}`}
                onClick={() => {
                  if (window.confirm(`Delete ${s.name}? This removes the server and its allow-list entry.`)) {
                    del.mutate(s.id);
                  }
                }}
              >
                <Trash2 className="h-4 w-4 text-critical" />
              </Button>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
