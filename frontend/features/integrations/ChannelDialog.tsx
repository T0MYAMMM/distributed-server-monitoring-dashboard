"use client";

import { useState } from "react";
import toast from "react-hot-toast";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAddChannel, useUpdateChannel } from "@/lib/hooks/useChannels";
import { channelDef } from "./catalog";
import type { ChannelType, NotificationChannel } from "@/lib/api/types";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  type?: ChannelType; // add mode
  channel?: NotificationChannel; // edit mode
}

// ChannelDialog adds or edits one notification channel. In edit mode, secret
// fields show a "kept" placeholder and are only sent when re-entered.
export function ChannelDialog({ open, onOpenChange, type, channel }: Props) {
  const editing = !!channel;
  const def = channelDef(channel?.type ?? type ?? "slack");
  const add = useAddChannel();
  const update = useUpdateChannel();
  const pending = add.isPending || update.isPending;

  const [name, setName] = useState(channel?.name ?? def.label);
  const [config, setConfig] = useState<Record<string, string>>(channel?.config ?? {});
  const [enabled, setEnabled] = useState(channel?.enabled ?? true);

  const secretsSet = new Set(channel?.secrets_set ?? []);

  const submit = () => {
    const payload = { name: name.trim() || def.label, config, enabled };
    const onError = (e: Error) => toast.error(e.message || "Failed to save channel");
    if (editing) {
      update.mutate(
        { id: channel!.id, ...payload },
        { onSuccess: () => (toast.success("Channel updated"), onOpenChange(false)), onError },
      );
    } else {
      add.mutate(
        { type: def.type, ...payload },
        { onSuccess: () => (toast.success("Channel added"), onOpenChange(false)), onError },
      );
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <def.icon className="h-5 w-5 text-primary" />
            {editing ? `Edit ${def.label}` : `Add ${def.label}`}
          </DialogTitle>
          <DialogDescription>{def.blurb}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="ch-name">Name</Label>
            <Input id="ch-name" value={name} onChange={(e) => setName(e.target.value)} placeholder={def.label} />
          </div>

          {def.fields.map((f) => (
            <div key={f.key} className="space-y-1.5">
              <Label htmlFor={`ch-${f.key}`}>
                {f.label}
                {f.optional && <span className="ml-1 text-xs text-muted-foreground">(optional)</span>}
              </Label>
              <Input
                id={`ch-${f.key}`}
                type={f.secret ? "password" : "text"}
                value={config[f.key] ?? ""}
                placeholder={
                  f.secret && secretsSet.has(f.key) ? "•••••• (leave blank to keep)" : f.placeholder
                }
                onChange={(e) => setConfig((c) => ({ ...c, [f.key]: e.target.value }))}
              />
              {f.help && <p className="text-xs text-muted-foreground">{f.help}</p>}
            </div>
          ))}

          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={enabled}
              onChange={(e) => setEnabled(e.target.checked)}
              className="h-4 w-4 rounded border-border accent-primary"
            />
            Enabled — receive live alerts
          </label>
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)} disabled={pending}>
            Cancel
          </Button>
          <Button onClick={submit} disabled={pending}>
            {editing ? "Save changes" : "Add channel"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
