"use client";

import { useEffect, useMemo, useState } from "react";
import toast from "react-hot-toast";
import { Lock, RotateCw, Save, SlidersHorizontal } from "lucide-react";
import { PageHeader } from "@/components/shared/PageHeader";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useSettings, useUpdateSettings } from "@/lib/hooks/useSettings";
import { useAuth } from "@/lib/hooks/useAuth";
import { cn } from "@/lib/utils";
import type { SettingField } from "@/lib/api/types";

const SECTION_ORDER = ["General", "Data & retention", "Thresholds", "Security"];

export default function SettingsPage() {
  const { data, isLoading } = useSettings();
  const update = useUpdateSettings();
  const { isAuthenticated } = useAuth();

  // Local edit buffer keyed by setting key; seeded from the server doc.
  const [edits, setEdits] = useState<Record<string, string>>({});
  useEffect(() => {
    if (data) setEdits(Object.fromEntries(data.fields.map((f) => [f.key, f.value])));
  }, [data]);

  const dirty = useMemo(() => {
    if (!data) return {};
    const out: Record<string, string> = {};
    for (const f of data.fields) {
      if (!f.env_locked && edits[f.key] !== undefined && edits[f.key] !== f.value) {
        out[f.key] = edits[f.key];
      }
    }
    return out;
  }, [data, edits]);
  const dirtyCount = Object.keys(dirty).length;

  const sections = useMemo(() => {
    const map = new Map<string, SettingField[]>();
    for (const f of data?.fields ?? []) {
      if (!map.has(f.section)) map.set(f.section, []);
      map.get(f.section)!.push(f);
    }
    return Array.from(map.entries()).sort(
      (a, b) => SECTION_ORDER.indexOf(a[0]) - SECTION_ORDER.indexOf(b[0]),
    );
  }, [data]);

  const onSave = () => {
    if (!isAuthenticated) {
      toast.error("Log in (Admin) to change settings.");
      return;
    }
    update.mutate(dirty, {
      onSuccess: () => toast.success("Settings saved"),
      onError: (e: Error) => toast.error(e.message || "Failed to save"),
    });
  };

  const set = (key: string, value: string) => setEdits((e) => ({ ...e, [key]: value }));

  return (
    <div className="space-y-6">
      <PageHeader
        title="Settings"
        description="Configuration that used to live only in environment variables — editable here, with env values kept as overrides."
        actions={
          <Button onClick={onSave} disabled={dirtyCount === 0 || update.isPending}>
            <Save className="h-4 w-4" />
            {dirtyCount > 0 ? `Save ${dirtyCount} change${dirtyCount === 1 ? "" : "s"}` : "Saved"}
          </Button>
        }
      />

      {isLoading && !data ? (
        <div className="space-y-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-48 w-full" />
          ))}
        </div>
      ) : (
        <>
          {sections.map(([section, fields]) => (
            <Card key={section}>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <SlidersHorizontal className="h-4 w-4 text-muted-foreground" />
                  {section}
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-5">
                {fields.map((f) => (
                  <Field key={f.key} field={f} value={edits[f.key] ?? f.value} onChange={(v) => set(f.key, v)} />
                ))}
              </CardContent>
            </Card>
          ))}

          {data?.about && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">About</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                <About label="Version" value={data.about.version} />
                <About label="Runtime" value={data.about.go} />
                <About label="Log database" value={data.about.log_database} />
                <About label="Instance" value={data.about.instance_name} />
              </CardContent>
            </Card>
          )}
        </>
      )}
    </div>
  );
}

function About({ label, value }: { label: string; value?: string }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-0.5 font-mono text-sm capitalize">{value ?? "—"}</p>
    </div>
  );
}

function Field({
  field,
  value,
  onChange,
}: {
  field: SettingField;
  value: string;
  onChange: (v: string) => void;
}) {
  const locked = field.env_locked;
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between sm:gap-8">
      <div className="min-w-0 sm:max-w-md">
        <div className="flex flex-wrap items-center gap-2">
          <label htmlFor={field.key} className="text-sm font-medium">
            {field.label}
          </label>
          {locked && (
            <Badge variant="muted" className="gap-1">
              <Lock className="h-3 w-3" /> {field.env_var}
            </Badge>
          )}
          {field.restart_required && (
            <Badge variant="outline" className="gap-1">
              <RotateCw className="h-3 w-3" /> restart
            </Badge>
          )}
        </div>
        {field.help && <p className="mt-1 text-sm text-muted-foreground">{field.help}</p>}
      </div>
      <div className="w-full shrink-0 sm:w-64">
        <Control field={field} value={value} onChange={onChange} disabled={locked} />
      </div>
    </div>
  );
}

function Control({
  field,
  value,
  onChange,
  disabled,
}: {
  field: SettingField;
  value: string;
  onChange: (v: string) => void;
  disabled: boolean;
}) {
  if (field.kind === "bool") {
    const on = value === "true";
    return (
      <button
        type="button"
        role="switch"
        aria-checked={on}
        aria-label={field.label}
        disabled={disabled}
        onClick={() => onChange(on ? "false" : "true")}
        className={cn(
          "relative inline-flex h-6 w-11 items-center rounded-full transition-colors disabled:cursor-not-allowed disabled:opacity-50",
          on ? "bg-primary" : "bg-muted",
        )}
      >
        <span
          className={cn(
            "inline-block h-4 w-4 transform rounded-full bg-background transition-transform",
            on ? "translate-x-6" : "translate-x-1",
          )}
        />
      </button>
    );
  }
  if (field.kind === "enum") {
    return (
      <select
        id={field.key}
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
        className="flex h-10 w-full rounded-md border border-input bg-background px-3 text-sm capitalize ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
      >
        {(field.options ?? []).map((o) => (
          <option key={o} value={o}>
            {o}
          </option>
        ))}
      </select>
    );
  }
  return (
    <Input
      id={field.key}
      type={field.kind === "int" || field.kind === "float" ? "number" : "text"}
      step={field.kind === "float" ? "0.1" : undefined}
      min={field.min}
      max={field.max}
      value={value}
      disabled={disabled}
      onChange={(e) => onChange(e.target.value)}
    />
  );
}
