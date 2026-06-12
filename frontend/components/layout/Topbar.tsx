"use client";

import Link from "next/link";
import { Bell, Globe, Menu, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ThemeToggle } from "./ThemeToggle";
import { useUnacknowledgedCount } from "@/lib/hooks/useAlerts";

export function Topbar({ onOpenMobile }: { onOpenMobile: () => void }) {
  const unacked = useUnacknowledgedCount();

  return (
    <header className="sticky top-0 z-30 flex h-16 items-center gap-3 border-b border-border bg-background/80 px-4 backdrop-blur">
      <button
        onClick={onOpenMobile}
        aria-label="Open menu"
        className="rounded-md p-2 text-muted-foreground hover:bg-accent lg:hidden"
      >
        <Menu className="h-5 w-5" />
      </button>

      {/* Global search hint; the command palette opens on Cmd/Ctrl+K. */}
      <button
        onClick={() => document.dispatchEvent(new KeyboardEvent("keydown", { key: "k", metaKey: true }))}
        className="flex h-9 max-w-md flex-1 items-center gap-2 rounded-md border border-border bg-card px-3 text-sm text-muted-foreground hover:bg-accent"
      >
        <Search className="h-4 w-4" />
        <span className="flex-1 text-left">Search…</span>
        <kbd className="hidden rounded border border-border bg-muted px-1.5 text-[10px] sm:inline">⌘K</kbd>
      </button>

      <div className="ml-auto flex items-center gap-1">
        <Link href="/alerts" aria-label="Alerts" className="relative">
          <Button variant="ghost" size="icon">
            <Bell className="h-5 w-5" />
          </Button>
          {unacked > 0 && (
            <span className="absolute right-1 top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-critical px-1 text-[10px] font-semibold text-critical-foreground">
              {unacked > 9 ? "9+" : unacked}
            </span>
          )}
        </Link>
        <ThemeToggle />
        <Button variant="ghost" size="icon" aria-label="Language" title="Language (coming soon)">
          <Globe className="h-5 w-5" />
        </Button>
      </div>
    </header>
  );
}
