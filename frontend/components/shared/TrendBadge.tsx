import { ArrowDownRight, ArrowUpRight, Minus } from "lucide-react";
import { cn } from "@/lib/utils";

interface TrendBadgeProps {
  delta: number;
  // Which direction is "good". For CPU/memory/disk a rise is bad (down is good);
  // for active-servers a rise is good (up).
  goodDirection?: "up" | "down";
  suffix?: string;
}

// TrendBadge shows the change versus the previous window, colored green when the
// movement is favorable and red when not.
export function TrendBadge({ delta, goodDirection = "down", suffix = "" }: TrendBadgeProps) {
  const rounded = Math.round(delta * 10) / 10;
  if (rounded === 0) {
    return (
      <span className="inline-flex items-center gap-0.5 text-xs font-medium text-muted-foreground">
        <Minus className="h-3 w-3" />
        0{suffix}
      </span>
    );
  }
  const isUp = rounded > 0;
  const isGood = (isUp && goodDirection === "up") || (!isUp && goodDirection === "down");
  const Icon = isUp ? ArrowUpRight : ArrowDownRight;
  return (
    <span
      className={cn(
        "inline-flex items-center gap-0.5 text-xs font-medium",
        isGood ? "text-success" : "text-critical",
      )}
    >
      <Icon className="h-3 w-3" />
      {isUp ? "+" : ""}
      {rounded}
      {suffix}
    </span>
  );
}
