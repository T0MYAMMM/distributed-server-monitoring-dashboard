"use client";

import { useState } from "react";
import toast from "react-hot-toast";
import { Bug, Heart, Lightbulb, MessageSquare, Send, Sparkles } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { PageHeader } from "@/components/shared/PageHeader";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useSubmitFeedback } from "@/lib/hooks/useFeedback";
import type { FeedbackCategory } from "@/lib/api/types";
import { cn } from "@/lib/utils";

const CATEGORIES: { value: FeedbackCategory; label: string; icon: LucideIcon }[] = [
  { value: "bug", label: "Bug", icon: Bug },
  { value: "idea", label: "Idea", icon: Lightbulb },
  { value: "praise", label: "Praise", icon: Heart },
  { value: "general", label: "General", icon: MessageSquare },
];

// What's new — plain, factual release notes (no hype).
const CHANGELOG: { version: string; date: string; notes: string[] }[] = [
  {
    version: "1.0",
    date: "June 2026",
    notes: [
      "Settings: instance name, thresholds, retention, and IP masking are now editable in-app.",
      "Integrations: Slack, Discord, ntfy, webhook, PagerDuty, and email alert channels, plus a log-source snippet generator.",
      "Analytics: fleet utilization, per-server uptime and disk-capacity projections, and log volume by level.",
      "Help: architecture, add-a-machine, the logs cookbook, and keyboard shortcuts, in-app and searchable.",
    ],
  },
  {
    version: "0.9",
    date: "June 2026",
    notes: [
      "Logs & Activity: per-node app/module multi-select and message-only grep.",
      "Drag-and-drop server ordering and the CloudGuard browser icon.",
    ],
  },
];

export default function FeedbackPage() {
  const [category, setCategory] = useState<FeedbackCategory>("idea");
  const [message, setMessage] = useState("");
  const submit = useSubmitFeedback();

  const onSubmit = () => {
    const text = message.trim();
    if (!text) {
      toast.error("Add a short message first.");
      return;
    }
    submit.mutate(
      { category, message: text, page: "feedback" },
      {
        onSuccess: () => {
          toast.success("Thanks — we've got it.");
          setMessage("");
          setCategory("idea");
        },
        onError: (e: Error) => toast.error(e.message || "Couldn't send that."),
      },
    );
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Feedback"
        description="Tell us what's working and what isn't. It goes straight to the team."
      />

      <div className="grid gap-6 lg:grid-cols-5">
        <Card className="lg:col-span-3">
          <CardHeader>
            <CardTitle className="text-base">Send feedback</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              {CATEGORIES.map((c) => (
                <button
                  key={c.value}
                  onClick={() => setCategory(c.value)}
                  className={cn(
                    "flex flex-col items-center gap-1.5 rounded-md border p-3 text-sm transition-colors",
                    category === c.value
                      ? "border-primary bg-primary/10 text-foreground"
                      : "border-border text-muted-foreground hover:bg-accent",
                  )}
                >
                  <c.icon className="h-4 w-4" />
                  {c.label}
                </button>
              ))}
            </div>

            <textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              rows={6}
              placeholder="What happened, or what would help? Specifics are welcome."
              className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            />

            <div className="flex items-center justify-between">
              <p className="text-xs text-muted-foreground">No account details are attached.</p>
              <Button onClick={onSubmit} disabled={submit.isPending}>
                <Send className="h-4 w-4" /> Send feedback
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Sparkles className="h-4 w-4 text-primary" /> What&apos;s new
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-5">
            {CHANGELOG.map((rel) => (
              <div key={rel.version}>
                <p className="text-sm font-medium">
                  v{rel.version}
                  <span className="ml-2 text-xs font-normal text-muted-foreground">{rel.date}</span>
                </p>
                <ul className="mt-1.5 space-y-1.5">
                  {rel.notes.map((n, i) => (
                    <li key={i} className="flex gap-2 text-sm text-muted-foreground">
                      <span className="mt-1.5 h-1 w-1 shrink-0 rounded-full bg-primary" />
                      {n}
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
