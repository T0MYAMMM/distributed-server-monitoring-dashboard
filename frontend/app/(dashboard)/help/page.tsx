"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { ArrowRight, Keyboard, Search } from "lucide-react";
import { PageHeader } from "@/components/shared/PageHeader";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { EmptyState } from "@/components/shared/EmptyState";
import { HELP_ARTICLES, SHORTCUTS } from "@/features/help/content";

export default function HelpPage() {
  const [q, setQ] = useState("");

  const articles = useMemo(() => {
    const needle = q.trim().toLowerCase();
    if (!needle) return HELP_ARTICLES;
    return HELP_ARTICLES.filter((a) =>
      [a.title, a.summary, ...(a.paragraphs ?? []), ...(a.steps ?? []), ...a.tags]
        .join(" ")
        .toLowerCase()
        .includes(needle),
    );
  }, [q]);

  return (
    <div className="space-y-6">
      <PageHeader title="Help" description="Runbooks and references for operating CloudGuard, in one place." />

      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search help…"
          className="pl-9"
        />
      </div>

      {articles.length === 0 ? (
        <EmptyState Icon={Search} title="No matches" description="Try a different term, like “agent”, “logs”, or “threshold”." />
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          {articles.map((a) => (
            <Card key={a.id}>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <a.icon className="h-4 w-4 text-primary" />
                  {a.title}
                </CardTitle>
                <p className="text-sm text-muted-foreground">{a.summary}</p>
              </CardHeader>
              <CardContent className="space-y-3 text-sm">
                {a.paragraphs?.map((p, i) => (
                  <p key={i} className="text-muted-foreground">
                    {p}
                  </p>
                ))}
                {a.steps && (
                  <ol className="list-decimal space-y-1.5 pl-5 text-muted-foreground">
                    {a.steps.map((s, i) => (
                      <li key={i}>{s}</li>
                    ))}
                  </ol>
                )}
                {a.code && (
                  <pre className="cg-logpane overflow-x-auto rounded-md border border-border bg-muted/40 p-3 text-xs leading-relaxed">
                    <code>{a.code}</code>
                  </pre>
                )}
                {a.link && (
                  <Link
                    href={a.link.href}
                    className="inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline"
                  >
                    {a.link.label} <ArrowRight className="h-3.5 w-3.5" />
                  </Link>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Keyboard className="h-4 w-4 text-primary" /> Keyboard shortcuts
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {SHORTCUTS.map((s) => (
            <div key={s.action} className="flex items-center justify-between gap-4 text-sm">
              <span className="text-muted-foreground">{s.action}</span>
              <span className="flex shrink-0 gap-1">
                {s.keys.map((k) => (
                  <kbd
                    key={k}
                    className="rounded border border-border bg-muted px-1.5 py-0.5 font-mono text-xs"
                  >
                    {k}
                  </kbd>
                ))}
              </span>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
