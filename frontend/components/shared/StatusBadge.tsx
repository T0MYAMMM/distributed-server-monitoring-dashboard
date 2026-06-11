import { Activity, CircleSlash, Clock, AlertTriangle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { ServerStatus } from "@/lib/api/types";

// StatusBadge renders a server's health. Color is never the only signal: each
// state carries an icon and label too.
const MAP: Record<
  ServerStatus,
  { label: string; variant: "success" | "critical" | "muted"; Icon: typeof Activity }
> = {
  running: { label: "Healthy", variant: "success", Icon: Activity },
  stopped: { label: "Critical", variant: "critical", Icon: CircleSlash },
  maintenance: { label: "Pending", variant: "muted", Icon: Clock },
};

export function StatusBadge({ status }: { status: ServerStatus }) {
  const { label, variant, Icon } = MAP[status] ?? {
    label: "Unknown",
    variant: "muted" as const,
    Icon: AlertTriangle,
  };
  return (
    <Badge variant={variant}>
      <Icon className="h-3 w-3" />
      {label}
    </Badge>
  );
}
