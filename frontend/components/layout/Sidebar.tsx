"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Activity, PanelLeftClose, PanelLeft, UserCircle2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { mainNav, supportNav, adminNavItem, type NavItem } from "@/config/nav";

function isActive(pathname: string, href: string): boolean {
  if (href === "/") return pathname === "/";
  return pathname === href || pathname.startsWith(`${href}/`);
}

function NavLink({ item, collapsed }: { item: NavItem; collapsed: boolean }) {
  const pathname = usePathname();
  const active = isActive(pathname, item.href);
  return (
    <Link
      href={item.href}
      title={item.title}
      className={cn(
        "group flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
        active
          ? "bg-primary text-primary-foreground"
          : "text-muted-foreground hover:bg-accent hover:text-foreground",
        collapsed && "justify-center px-2",
      )}
    >
      <item.icon className="h-[18px] w-[18px] shrink-0" />
      {!collapsed && <span className="flex-1 truncate">{item.title}</span>}
      {!collapsed && item.comingSoon && (
        <span className="rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">Soon</span>
      )}
    </Link>
  );
}

export function Sidebar({
  collapsed,
  onToggleCollapse,
}: {
  collapsed: boolean;
  onToggleCollapse: () => void;
}) {
  return (
    <div className="flex h-full flex-col bg-card">
      {/* Brand + collapse toggle */}
      <div className={cn("flex h-16 items-center gap-2 border-b border-border px-4", collapsed && "justify-center px-2")}>
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
          <Activity className="h-5 w-5" />
        </span>
        {!collapsed && <span className="text-base font-semibold tracking-tight">CloudGuard</span>}
        <button
          onClick={onToggleCollapse}
          aria-label="Toggle sidebar"
          className="ml-auto hidden rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground lg:block"
        >
          {collapsed ? <PanelLeft className="h-4 w-4" /> : <PanelLeftClose className="h-4 w-4" />}
        </button>
      </div>

      <nav className="flex-1 space-y-6 overflow-y-auto px-3 py-4">
        <div className="space-y-1">
          {!collapsed && <p className="px-3 pb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">Main Navigation</p>}
          {mainNav.map((item) => (
            <NavLink key={item.href} item={item} collapsed={collapsed} />
          ))}
        </div>
        <div className="space-y-1">
          {!collapsed && <p className="px-3 pb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">Support</p>}
          {supportNav.map((item) => (
            <NavLink key={item.href} item={item} collapsed={collapsed} />
          ))}
          <NavLink item={adminNavItem} collapsed={collapsed} />
        </div>
      </nav>

      {/* Account card pinned at the bottom */}
      <div className="border-t border-border p-3">
        <Link
          href="/admin"
          className={cn(
            "flex items-center gap-3 rounded-md px-2 py-2 text-sm hover:bg-accent",
            collapsed && "justify-center",
          )}
        >
          <UserCircle2 className="h-7 w-7 shrink-0 text-muted-foreground" />
          {!collapsed && (
            <div className="min-w-0">
              <p className="truncate font-medium">Admin</p>
              <p className="truncate text-xs text-muted-foreground">Manage fleet</p>
            </div>
          )}
        </Link>
      </div>
    </div>
  );
}
