import type { LucideIcon } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { TrendBadge } from "./TrendBadge";

interface StatCardProps {
  label: string;
  value: string;
  Icon: LucideIcon;
  delta?: number;
  deltaSuffix?: string;
  goodDirection?: "up" | "down";
  loading?: boolean;
}

// StatCard is the shared KPI-card anatomy: icon + label, large metric, footer
// trend badge. Every dashboard metric card is built from this.
export function StatCard({
  label,
  value,
  Icon,
  delta,
  deltaSuffix,
  goodDirection,
  loading,
}: StatCardProps) {
  return (
    <Card>
      <CardContent className="p-5">
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">{label}</span>
          <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <Icon className="h-5 w-5" />
          </span>
        </div>
        {loading ? (
          <Skeleton className="mt-3 h-8 w-24" />
        ) : (
          <div className="mt-3 text-2xl font-semibold tracking-tight">{value}</div>
        )}
        {delta !== undefined && !loading && (
          <div className="mt-2 flex items-center gap-1.5">
            <TrendBadge delta={delta} suffix={deltaSuffix} goodDirection={goodDirection} />
            <span className="text-xs text-muted-foreground">vs prev window</span>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
