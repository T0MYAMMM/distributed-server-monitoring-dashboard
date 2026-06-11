"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Command } from "cmdk";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { mainNav, supportNav, adminNavItem } from "@/config/nav";
import { useServers } from "@/lib/hooks/useServers";

// CommandPalette (Cmd/Ctrl+K) navigates pages and jumps to a server by name.
export function CommandPalette() {
  const [open, setOpen] = useState(false);
  const router = useRouter();
  const { data: servers } = useServers();

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((o) => !o);
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  const go = (href: string) => {
    setOpen(false);
    router.push(href);
  };

  const navItems = [...mainNav, ...supportNav, adminNavItem];

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="max-w-xl overflow-hidden p-0">
        <Command className="[&_[cmdk-input]]:h-12 [&_[cmdk-input]]:w-full [&_[cmdk-input]]:bg-transparent [&_[cmdk-input]]:px-4 [&_[cmdk-input]]:text-sm [&_[cmdk-input]]:outline-none">
          <Command.Input placeholder="Search pages and servers…" autoFocus />
          <Command.List className="max-h-80 overflow-y-auto border-t border-border p-2">
            <Command.Empty className="px-3 py-6 text-center text-sm text-muted-foreground">
              No results found.
            </Command.Empty>
            <Command.Group heading="Navigation" className="px-1 py-1 text-xs text-muted-foreground">
              {navItems.map((item) => (
                <Command.Item
                  key={item.href}
                  value={`nav ${item.title}`}
                  onSelect={() => go(item.href)}
                  className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-2 text-sm text-foreground aria-selected:bg-accent"
                >
                  <item.icon className="h-4 w-4 text-muted-foreground" />
                  {item.title}
                </Command.Item>
              ))}
            </Command.Group>
            {servers && servers.length > 0 && (
              <Command.Group heading="Servers" className="px-1 py-1 text-xs text-muted-foreground">
                {servers.map((s) => (
                  <Command.Item
                    key={s.id}
                    value={`server ${s.name}`}
                    onSelect={() => go(`/servers?focus=${encodeURIComponent(s.id)}`)}
                    className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-2 text-sm text-foreground aria-selected:bg-accent"
                  >
                    {s.name}
                  </Command.Item>
                ))}
              </Command.Group>
            )}
          </Command.List>
        </Command>
      </DialogContent>
    </Dialog>
  );
}
