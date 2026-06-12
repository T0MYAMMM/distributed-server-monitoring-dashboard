"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { LogOut, Server as ServerIcon, Activity, Clock, CircleSlash } from "lucide-react";
import { PageHeader } from "@/components/shared/PageHeader";
import { StatCard } from "@/components/shared/StatCard";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { AddClientDialog } from "@/features/admin/AddClientDialog";
import { ResetPasswordDialog } from "@/features/admin/ResetPasswordDialog";
import { AdminClientTable } from "@/features/admin/AdminClientTable";
import { UnknownAgentsCard } from "@/features/admin/UnknownAgentsCard";
import { useServers } from "@/lib/hooks/useServers";
import { useAuth } from "@/lib/hooks/useAuth";

export default function AdminPage() {
  const router = useRouter();
  const { isAuthenticated, ready, logout } = useAuth();
  const { data: servers, isLoading } = useServers();

  useEffect(() => {
    if (ready && !isAuthenticated) router.replace("/login");
  }, [ready, isAuthenticated, router]);

  if (!ready || !isAuthenticated) return null;

  const list = servers ?? [];
  const active = list.filter((s) => s.status === "running").length;
  const pending = list.filter((s) => s.status === "maintenance").length;
  const stopped = list.filter((s) => s.status === "stopped").length;

  const onLogout = () => {
    logout();
    router.replace("/login");
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Client Management"
        description="Register machines, copy install commands, and manage the fleet."
        actions={
          <>
            <ResetPasswordDialog />
            <AddClientDialog />
            <Button variant="ghost" onClick={onLogout}>
              <LogOut className="h-4 w-4" /> Logout
            </Button>
          </>
        }
      />

      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <StatCard label="Total Clients" Icon={ServerIcon} loading={isLoading} value={String(list.length)} />
        <StatCard label="Active" Icon={Activity} loading={isLoading} value={String(active)} />
        <StatCard label="Pending" Icon={Clock} loading={isLoading} value={String(pending)} />
        <StatCard label="Stopped" Icon={CircleSlash} loading={isLoading} value={String(stopped)} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Clients</CardTitle>
        </CardHeader>
        <CardContent className="px-0 pb-0">
          <AdminClientTable servers={servers} />
        </CardContent>
      </Card>

      <UnknownAgentsCard />
    </div>
  );
}
