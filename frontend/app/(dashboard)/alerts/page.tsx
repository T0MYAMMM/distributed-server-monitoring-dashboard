"use client";

import { useState } from "react";
import toast from "react-hot-toast";
import { BellOff, Check } from "lucide-react";
import { PageHeader } from "@/components/shared/PageHeader";
import { EmptyState } from "@/components/shared/EmptyState";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useAlerts, useAcknowledgeAlert } from "@/lib/hooks/useAlerts";
import { useAuth } from "@/lib/hooks/useAuth";
import type { AlertSeverity } from "@/lib/api/types";

const FILTERS: { label: string; value?: AlertSeverity }[] = [
  { label: "All" },
  { label: "Critical", value: "critical" },
  { label: "Warning", value: "warning" },
  { label: "Info", value: "info" },
];

const SEVERITY_VARIANT: Record<AlertSeverity, "critical" | "warning" | "default"> = {
  critical: "critical",
  warning: "warning",
  info: "default",
};

export default function AlertsPage() {
  const [severity, setSeverity] = useState<AlertSeverity | undefined>(undefined);
  const { data, isLoading } = useAlerts(severity);
  const ack = useAcknowledgeAlert();
  const { isAuthenticated } = useAuth();
  const alerts = data ?? [];

  const onAck = (id: number) => {
    if (!isAuthenticated) {
      toast.error("Log in (Admin) to acknowledge alerts.");
      return;
    }
    ack.mutate(id, {
      onSuccess: () => toast.success("Alert acknowledged"),
      onError: () => toast.error("Failed to acknowledge"),
    });
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Alerts & Incidents"
        description="Status changes and threshold breaches across the fleet."
        actions={
          <div className="flex gap-1">
            {FILTERS.map((f) => (
              <Button
                key={f.label}
                size="sm"
                variant={severity === f.value ? "secondary" : "ghost"}
                onClick={() => setSeverity(f.value)}
              >
                {f.label}
              </Button>
            ))}
          </div>
        }
      />

      {isLoading && alerts.length === 0 ? (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      ) : alerts.length === 0 ? (
        <EmptyState Icon={BellOff} title="No alerts" description="Nothing to report. New incidents will appear here." />
      ) : (
        <div className="space-y-2">
          {alerts.map((a) => {
            const acked = a.acknowledged_at !== "";
            return (
              <Card key={a.id} className={acked ? "opacity-60" : ""}>
                <CardContent className="flex items-center gap-4 p-4">
                  <Badge variant={SEVERITY_VARIANT[a.severity]}>{a.severity}</Badge>
                  <div className="min-w-0 flex-1">
                    <p className="truncate font-medium">{a.message}</p>
                    <p className="text-xs text-muted-foreground">
                      {a.server_name} · {a.created_at}
                      {acked && ` · acknowledged ${a.acknowledged_at}`}
                    </p>
                  </div>
                  {acked ? (
                    <span className="flex items-center gap-1 text-sm text-success">
                      <Check className="h-4 w-4" /> Acknowledged
                    </span>
                  ) : (
                    <Button size="sm" variant="outline" onClick={() => onAck(a.id)} disabled={ack.isPending}>
                      Acknowledge
                    </Button>
                  )}
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
