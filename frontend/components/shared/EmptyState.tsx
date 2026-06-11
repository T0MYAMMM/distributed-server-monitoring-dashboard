import type { LucideIcon } from "lucide-react";

interface EmptyStateProps {
  Icon: LucideIcon;
  title: string;
  description?: string;
  action?: React.ReactNode;
}

// EmptyState is the shared empty/zero-data placeholder for lists and panels.
export function EmptyState({ Icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border px-6 py-12 text-center">
      <span className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
        <Icon className="h-6 w-6" />
      </span>
      <div>
        <p className="font-medium">{title}</p>
        {description && <p className="mt-1 text-sm text-muted-foreground">{description}</p>}
      </div>
      {action}
    </div>
  );
}
