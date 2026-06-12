import {
  LayoutDashboard,
  Server,
  Bell,
  ScrollText,
  BarChart3,
  Plug,
  MessageSquare,
  HelpCircle,
  Settings,
  ShieldCheck,
  type LucideIcon,
} from "lucide-react";

// Sidebar items are data. Adding a feature page = add a route + one entry here.
export interface NavItem {
  title: string;
  href: string;
  icon: LucideIcon;
  comingSoon?: boolean;
}

export const mainNav: NavItem[] = [
  { title: "Dashboard", href: "/", icon: LayoutDashboard },
  { title: "Servers", href: "/servers", icon: Server },
  { title: "Alerts & Incidents", href: "/alerts", icon: Bell },
  { title: "Logs & Activity", href: "/logs", icon: ScrollText },
  { title: "Analytics", href: "/analytics", icon: BarChart3, comingSoon: true },
  { title: "Integrations", href: "/integrations", icon: Plug, comingSoon: true },
];

export const supportNav: NavItem[] = [
  { title: "Feedback", href: "/feedback", icon: MessageSquare, comingSoon: true },
  { title: "Help", href: "/help", icon: HelpCircle, comingSoon: true },
  { title: "Settings", href: "/settings", icon: Settings, comingSoon: true },
];

export const adminNavItem: NavItem = { title: "Admin", href: "/admin", icon: ShieldCheck };
