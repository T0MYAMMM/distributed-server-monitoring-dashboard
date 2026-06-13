# CloudGuard — Design System

Private fleet observability — **metrics, alerts, and logs in one pane**, riding
entirely on a Tailscale tailnet. This design system formalizes the visual
language of the shipped CloudGuard dashboard (dark-first, one orange accent,
8px rhythm, monospace log surface) into reusable tokens, components, foundation
specimens, and a full interactive UI kit.

> Mirrored from the claude.ai/design project **CloudGuard Design System**
> (`a66aa9b8-1177-47bf-a40f-f0973e6cf51e`). Refresh with the DesignSync tool /
> `/design-sync`. The local mirror tracks the foundation (tokens, assets,
> styles) + UI-kit; the full component variants + foundation specimens live in
> the live project.

> **Brand promise:** *See everything, privately, in two minutes.*
> **Pillars:** Private by default · Effortless onboarding · Calm clarity · Zero-dependency.

---

## Content fundamentals (voice & tone)

The voice is **calm, technical, trustworthy, plain-spoken**. Confident and quiet —
no growth-hack hype, no fear-based alerting language.

- **Person.** Address the operator directly and warmly where it's human
  ("Welcome back, Admin"); otherwise state facts plainly ("All systems nominal.",
  "Logs are not enabled").
- **Casing.** Sentence case for descriptions and helper text; Title Case for nav
  items and page titles ("Alerts & Incidents", "Logs & Activity"). Uppercase only
  for small section eyebrows ("MAIN NAVIGATION") and log levels.
- **Status words are operator-facing.** The backend lifecycle
  `running / stopped / maintenance` surfaces as **Healthy / Critical / Pending**.
- **Numbers carry units, inline.** "Global uptime 99.98% over the last 24h",
  "Disk > 90% on db-1", "CPU 88% sustained over 10m".
- **Empty/error states are kind and actionable** — name what's missing and the one
  next step.
- **No emoji** in product copy (only country flags in the server table, 🌐 fallback).
- **Microcopy is short.** Buttons are verbs ("Add Server", "Acknowledge",
  "Export CSV"). No trailing punctuation on buttons; ellipsis on deferred actions.

---

## Visual foundations

**Theme.** Dark-first. `:root`/`.dark` is canonical dark; `.light` derives from
the same token names.

**Color.** One warm **orange accent** (`#f97316` / `24 95% 53%`) used sparingly —
primary buttons, active nav, focus rings, the log module column, KPI icon tiles
(10% tint). Everything else is a tight blue-grey neutral ramp (`222 16–24%`).
Semantic status is **success (green) / warning (amber) / critical (red)**, each
solid and at a 15% tint. Charts use a 6-hue **data-viz series palette** (series-1
= brand orange) that stays distinguishable on dark.

**Status is never color-only** — every status carries an icon + text label.

**Typography.** **Geist** for UI, **Geist Mono** for the log/metric/IP surface.
One compact scale (14px body). Tabular figures on number columns. Tight heading
tracking (`-0.02em`).

**Spacing & radii.** 8px rhythm with 2/4px half-steps. Radius anchored on **10px**
(`--radius`): 6px chips, 8px buttons/inputs, 10px cards, pill for badges/gauges.
A **density toggle** (`.density-compact`) shrinks rows 48→36px.

**Surfaces & depth.** Layered surface ramp + 1px borders, not heavy shadows.
Shadows reserved for menus (md) and dialogs/drawers (lg).

**Backgrounds.** Flat dark surfaces — no gradients, textures, or illustrations.
Only imagery: the logo mark and country-flag glyphs.

**Motion.** Subtle, 120–280ms, ease-out. `prefers-reduced-motion` zeroes it.

**Interaction.** Hover = step up the surface ramp / accent darken; focus-visible =
2px accent ring; disabled = 50% opacity.

---

## Iconography

**[lucide](https://lucide.dev)** (`lucide-react`) exclusively — 24×24, 2px stroke,
rounded caps. No custom icon font. Import `lucide-react` in production (the UI kit
ships a lucide-equivalent `CGIcon` set for the standalone preview). Size: 16px in
buttons/badges, 18px nav, 20px KPI tiles, 24px empty-state tiles. **Brand assets**
in `assets/` (mark, wordmarks, mono mark) — the pulse = "your fleet's vitals."

---

## Index

**Foundations (root)**
- `styles.css` — global entry point (import list). Consumers link this.
- `components.css` — component class styles.
- `tokens/colors.css · typography.css · spacing.css · mono-surface.css · fonts.css`
- `assets/` — logo mark, wordmarks, monochrome mark.
- `foundations/*.html` — specimen cards (in the live project).

**Components** (`components/<group>/`)
| Group | Components |
|---|---|
| `core/` | `Button`, `Badge`, `Card` (+ Header/Title/Description/Content/Footer) |
| `status/` | `StatusBadge`, `TrendBadge`, `ResourceBar` |
| `data/` | `StatCard`, `Sparkline` |
| `forms/` | `Input` |
| `feedback/` | `EmptyState` |

Each has a `.jsx`, a `.d.ts` (props + usage), and a `.prompt.md`.

**UI kit** — `ui_kits/dashboard/`: interactive cockpit (Dashboard, Servers +
**Server detail drawer**, **Logs insights**, Alerts, ⌘K palette).

**Skill** — `SKILL.md`: entry point for using this system as a Claude Agent Skill.
