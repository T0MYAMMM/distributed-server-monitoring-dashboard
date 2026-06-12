"use client";

import { PageHeader } from "@/components/shared/PageHeader";
import { LogViewer } from "@/features/logs/LogViewer";

export default function LogsPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="Logs & Activity"
        description="Tail and search log files shipped by each VM's agent (stored in the external log database)."
      />
      <LogViewer />
    </div>
  );
}
