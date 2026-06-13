---
name: cloudguard-design
description: Use this skill to generate well-branded interfaces and assets for CloudGuard (private fleet observability — metrics, alerts, and logs on a Tailscale tailnet), either for production or throwaway prototypes/mocks. Contains essential design guidelines, colors, type, fonts, assets, and UI kit components for prototyping.
user-invocable: true
---

# CloudGuard design skill

Read `readme.md` first — it is the full design guide (sources, voice & tone,
visual foundations, iconography, and a manifest of everything here). Then explore
the files you need.

## What's here
- `styles.css` — the one global stylesheet to link. Tokens + component classes
  resolve through it. Never hardcode hex; use the `--cg-*` tokens (or the
  `hsl(var(--token))` shadcn channels).
- `tokens/` — colors, typography, spacing/radii/shadows/motion, the mono log
  surface, and font loading.
- `components/` — React primitives (Button, Badge, Card, StatusBadge, TrendBadge,
  ResourceBar, StatCard, Sparkline, Input, EmptyState). Each has a `.d.ts` +
  `.prompt.md`. Reachable at runtime via `window.CloudGuardDesignSystem_a66aa9`
  after loading `_ds_bundle.js`.
- `foundations/` — specimen cards for color, type, spacing, and brand.
- `assets/` — logo mark, wordmarks, monochrome mark (copy these out; don't redraw).
- `ui_kits/dashboard/` — a full interactive recreation of the fleet cockpit to
  reference for layout, density, and composition.

## How to work
- **Visual artifacts (slides, mocks, throwaway prototypes):** copy the assets and
  `styles.css` you need into a new folder and write static HTML. Mount components
  by loading `_ds_bundle.js` and reading `window.CloudGuardDesignSystem_a66aa9`,
  or just reuse the component CSS classes (`.cg-btn`, `.cg-badge`, `.cg-card`,
  `.cg-logpane`, …) directly. Use lucide for icons.
- **Production code:** the source app is Next.js + Tailwind + shadcn/Radix,
  dark-first, all color via tokens. Read `readme.md` and the token files to become
  an expert, then build in that stack. Import `lucide-react` for icons.

## Non-negotiables
- Dark-first, one orange accent used sparingly; neutral blue-grey ramp otherwise.
- **Status is never color-only** — always icon + label + color.
- Monospace (Geist Mono) for logs, metrics, IPs, IDs, and commands.
- Keyboard navigable, visible focus rings, 8px rhythm, calm/plain-spoken copy,
  no emoji (except country flags), no gradients/illustrations.

If invoked without guidance, ask what the user wants to build, ask a few focused
questions (surface, audience, production vs throwaway, variations), then act as an
expert CloudGuard designer producing HTML artifacts or production code.
