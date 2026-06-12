"use client";

import { useState } from "react";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";
import { Sidebar } from "@/components/layout/Sidebar";
import { Topbar } from "@/components/layout/Topbar";
import { CommandPalette } from "@/components/layout/CommandPalette";

// AppShell: fixed left sidebar + top bar + scrollable content. The sidebar
// collapses to an icon rail on demand and slides over on narrow viewports.
export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);

  return (
    <div className="flex min-h-screen">
      {/* Desktop sidebar */}
      <aside
        className={cn(
          "hidden shrink-0 border-r border-border transition-[width] duration-200 lg:block",
          collapsed ? "w-16" : "w-64",
        )}
      >
        <div className="fixed h-screen" style={{ width: collapsed ? "4rem" : "16rem" }}>
          <Sidebar collapsed={collapsed} onToggleCollapse={() => setCollapsed((c) => !c)} />
        </div>
      </aside>

      {/* Mobile off-canvas sidebar */}
      {mobileOpen && (
        <div className="fixed inset-0 z-50 lg:hidden">
          <div className="absolute inset-0 bg-black/60" onClick={() => setMobileOpen(false)} />
          <div className="absolute left-0 top-0 h-full w-64">
            <Sidebar collapsed={false} onToggleCollapse={() => setMobileOpen(false)} />
            <button
              onClick={() => setMobileOpen(false)}
              aria-label="Close menu"
              className="absolute right-2 top-4 rounded-md p-2 text-muted-foreground hover:bg-accent"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        </div>
      )}

      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar onOpenMobile={() => setMobileOpen(true)} />
        <main className="flex-1 overflow-x-hidden p-4 sm:p-6">{children}</main>
      </div>

      <CommandPalette />
    </div>
  );
}
