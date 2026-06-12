import { Construction } from "lucide-react";
import { PageHeader } from "./PageHeader";
import { EmptyState } from "./EmptyState";

// ComingSoon is the placeholder for feature pages that are not built yet, so the
// navigation can list every planned area.
export function ComingSoon({ title }: { title: string }) {
  return (
    <div className="space-y-6">
      <PageHeader title={title} description="This area is coming soon." />
      <EmptyState
        Icon={Construction}
        title={`${title} is on the roadmap`}
        description="The navigation lists every planned feature; this page will light up in a future release."
      />
    </div>
  );
}
