import { Bell, Hash, Mail, MessageSquare, Siren, Webhook, type LucideIcon } from "lucide-react";
import type { ChannelType } from "@/lib/api/types";

// One field of a channel's configuration form. Secret fields are masked on read
// and may be left blank on edit to keep the stored value.
export interface ChannelFieldDef {
  key: string;
  label: string;
  placeholder?: string;
  secret?: boolean;
  optional?: boolean;
  help?: string;
}

// A channel type the operator can add, with the fields its delivery needs.
export interface ChannelTypeDef {
  type: ChannelType;
  label: string;
  blurb: string;
  icon: LucideIcon;
  fields: ChannelFieldDef[];
}

export const CHANNEL_CATALOG: ChannelTypeDef[] = [
  {
    type: "slack",
    label: "Slack",
    blurb: "Post alerts to a channel via an incoming webhook.",
    icon: Hash,
    fields: [
      { key: "webhook_url", label: "Incoming webhook URL", secret: true, placeholder: "https://hooks.slack.com/services/…" },
    ],
  },
  {
    type: "discord",
    label: "Discord",
    blurb: "Send alerts to a Discord channel webhook.",
    icon: MessageSquare,
    fields: [{ key: "webhook_url", label: "Webhook URL", secret: true, placeholder: "https://discord.com/api/webhooks/…" }],
  },
  {
    type: "ntfy",
    label: "ntfy",
    blurb: "Push notifications to an ntfy topic, including self-hosted.",
    icon: Bell,
    fields: [
      { key: "url", label: "Topic URL", secret: true, placeholder: "https://ntfy.sh/my-topic" },
      { key: "token", label: "Access token", secret: true, optional: true, help: "Only for protected topics." },
    ],
  },
  {
    type: "webhook",
    label: "Generic webhook",
    blurb: "POST the raw alert JSON to any endpoint.",
    icon: Webhook,
    fields: [{ key: "url", label: "POST URL", secret: true, placeholder: "https://example.com/hooks/cloudguard" }],
  },
  {
    type: "pagerduty",
    label: "PagerDuty",
    blurb: "Trigger incidents through the Events API v2.",
    icon: Siren,
    fields: [{ key: "routing_key", label: "Integration routing key", secret: true }],
  },
  {
    type: "email",
    label: "Email (SMTP)",
    blurb: "Send alert emails through your SMTP relay.",
    icon: Mail,
    fields: [
      { key: "host", label: "SMTP host", placeholder: "smtp.example.com" },
      { key: "port", label: "Port", placeholder: "587" },
      { key: "from", label: "From address", placeholder: "cloudguard@example.com" },
      { key: "to", label: "Recipient(s)", placeholder: "ops@example.com, oncall@example.com" },
      { key: "username", label: "Username", optional: true },
      { key: "password", label: "Password", secret: true, optional: true },
    ],
  },
];

export function channelDef(type: ChannelType): ChannelTypeDef {
  return CHANNEL_CATALOG.find((c) => c.type === type) ?? CHANNEL_CATALOG[0];
}
