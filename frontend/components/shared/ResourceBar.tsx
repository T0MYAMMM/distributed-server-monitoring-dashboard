import { cn } from "@/lib/utils";
import { formatPercent, usageTone } from "@/lib/format";

const TONE: Record<string, string> = {
  success: "bg-success",
  warning: "bg-warning",
  critical: "bg-critical",
};

interface ResourceBarProps {
  label?: string;
  value: number; // 0-100
  showValue?: boolean;
  className?: string;
}

// ResourceBar is a segmented usage bar colored by threshold (green/amber/red).
export function ResourceBar({ label, value, showValue = true, className }: ResourceBarProps) {
  const pct = Math.max(0, Math.min(100, value ?? 0));
  return (
    <div className={cn("w-full", className)}>
      {(label || showValue) && (
        <div className="mb-1 flex items-center justify-between text-xs">
          {label && <span className="text-muted-foreground">{label}</span>}
          {showValue && <span className="font-medium tabular-nums">{formatPercent(pct, 0)}</span>}
        </div>
      )}
      <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
        <div className={cn("h-full rounded-full transition-all", TONE[usageTone(pct)])} style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}
