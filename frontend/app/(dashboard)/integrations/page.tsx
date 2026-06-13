"use client";

import { useState } from "react";
import toast from "react-hot-toast";
import { Lock, Plus, Send, Trash2 } from "lucide-react";
import { PageHeader } from "@/components/shared/PageHeader";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuth } from "@/lib/hooks/useAuth";
import {
  useChannels,
  useDeleteChannel,
  useTestChannel,
  useUpdateChannel,
} from "@/lib/hooks/useChannels";
import { cn } from "@/lib/utils";
import { CHANNEL_CATALOG, channelDef } from "@/features/integrations/catalog";
import { ChannelDialog } from "@/features/integrations/ChannelDialog";
import { LogSourceGenerator } from "@/features/integrations/LogSourceGenerator";
import type { ChannelType, NotificationChannel } from "@/lib/api/types";

export default function IntegrationsPage() {
  const { isAuthenticated } = useAuth();
  const { data: channels, isLoading } = useChannels(isAuthenticated);
  const toggle = useUpdateChannel();
  const test = useTestChannel();
  const del = useDeleteChannel();

  const [dialog, setDialog] = useState<
    { mode: "add"; type: ChannelType } | { mode: "edit"; channel: NotificationChannel } | null
  >(null);

  const onToggle = (c: NotificationChannel) =>
    toggle.mutate(
      { id: c.id, name: c.name, config: c.config, enabled: !c.enabled },
      { onError: (e: Error) => toast.error(e.message) },
    );

  const onTest = (c: NotificationChannel) =>
    test.mutate(c.id, {
      onSuccess: (r) =>
        r.ok ? toast.success(`Test sent to ${c.name}`) : toast.error(r.error || "Delivery failed"),
      onError: (e: Error) => toast.error(e.message),
    });

  const onDelete = (c: NotificationChannel) => {
    if (!confirm(`Delete the ${c.name} channel?`)) return;
    del.mutate(c.id, {
      onSuccess: () => toast.success("Channel deleted"),
      onError: (e: Error) => toast.error(e.message),
    });
  };

  return (
    <div className="space-y-8">
      <PageHeader
        title="Integrations"
        description="Connect outbound alert channels and wire a node's logs into CloudGuard."
      />

      {/* Alert channels */}
      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-medium">Alert channels</h2>
        </div>

        {!isAuthenticated ? (
          <Card>
            <CardContent className="flex items-center gap-3 p-4 text-sm text-muted-foreground">
              <Lock className="h-4 w-4" />
              Sign in as Admin to add and manage notification channels.
            </CardContent>
          </Card>
        ) : isLoading ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-32 w-full" />
            ))}
          </div>
        ) : (channels ?? []).length > 0 ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {channels!.map((c) => {
              const def = channelDef(c.type);
              return (
                <Card key={c.id}>
                  <CardContent className="space-y-3 p-4">
                    <div className="flex items-start justify-between gap-2">
                      <div className="flex items-center gap-2">
                        <span className="flex h-9 w-9 items-center justify-center rounded-md bg-primary/10 text-primary">
                          <def.icon className="h-4 w-4" />
                        </span>
                        <div>
                          <p className="font-medium leading-tight">{c.name}</p>
                          <p className="text-xs text-muted-foreground">{def.label}</p>
                        </div>
                      </div>
                      <button
                        type="button"
                        role="switch"
                        aria-checked={c.enabled}
                        aria-label="Enabled"
                        onClick={() => onToggle(c)}
                        className={cn(
                          "relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors",
                          c.enabled ? "bg-primary" : "bg-muted",
                        )}
                      >
                        <span
                          className={cn(
                            "inline-block h-4 w-4 transform rounded-full bg-background transition-transform",
                            c.enabled ? "translate-x-6" : "translate-x-1",
                          )}
                        />
                      </button>
                    </div>

                    <DeliveryStatus channel={c} />

                    <div className="flex gap-2">
                      <Button size="sm" variant="outline" onClick={() => onTest(c)} disabled={test.isPending}>
                        <Send className="h-3.5 w-3.5" /> Test
                      </Button>
                      <Button size="sm" variant="ghost" onClick={() => setDialog({ mode: "edit", channel: c })}>
                        Edit
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        className="ml-auto text-muted-foreground hover:text-critical"
                        onClick={() => onDelete(c)}
                        aria-label="Delete channel"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            No channels yet. Add one below — alerts will fan out to every enabled channel.
          </p>
        )}

        {/* Catalog */}
        <div>
          <p className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">Add a channel</p>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {CHANNEL_CATALOG.map((def) => (
              <button
                key={def.type}
                disabled={!isAuthenticated}
                onClick={() => setDialog({ mode: "add", type: def.type })}
                className="group flex items-start gap-3 rounded-lg border border-border p-4 text-left transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-50"
              >
                <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground group-hover:text-primary">
                  <def.icon className="h-4 w-4" />
                </span>
                <span className="min-w-0">
                  <span className="flex items-center gap-1.5 font-medium">
                    {def.label}
                    <Plus className="h-3.5 w-3.5 text-muted-foreground" />
                  </span>
                  <span className="mt-0.5 block text-xs text-muted-foreground">{def.blurb}</span>
                </span>
              </button>
            ))}
          </div>
        </div>
      </section>

      {/* Log sources */}
      <section className="space-y-4">
        <h2 className="text-lg font-medium">Log sources</h2>
        <LogSourceGenerator />
      </section>

      {dialog && (
        <ChannelDialog
          open
          onOpenChange={(o) => !o && setDialog(null)}
          type={dialog.mode === "add" ? dialog.type : undefined}
          channel={dialog.mode === "edit" ? dialog.channel : undefined}
        />
      )}
    </div>
  );
}

function DeliveryStatus({ channel }: { channel: NotificationChannel }) {
  if (channel.last_status === "ok") {
    return (
      <p className="text-xs">
        <Badge variant="success">Delivered</Badge>
        <span className="ml-2 text-muted-foreground">{channel.last_delivery}</span>
      </p>
    );
  }
  if (channel.last_status === "error") {
    return (
      <p className="text-xs">
        <Badge variant="critical">Failed</Badge>
        <span className="ml-2 truncate text-muted-foreground">{channel.last_error}</span>
      </p>
    );
  }
  return <p className="text-xs text-muted-foreground">No deliveries yet.</p>;
}
