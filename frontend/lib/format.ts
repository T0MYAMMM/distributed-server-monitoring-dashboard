// Display formatting helpers shared across dashboard widgets.

// formatBytesPerSec turns a bytes/second rate into a human string.
export function formatBytesPerSec(bytesPerSec: number): string {
  if (!bytesPerSec || bytesPerSec < 1) return "0 B/s";
  const units = ["B/s", "KB/s", "MB/s", "GB/s"];
  let v = bytesPerSec;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i += 1;
  }
  return `${v.toFixed(v >= 10 || i === 0 ? 0 : 1)} ${units[i]}`;
}

export function formatPercent(n: number, digits = 1): string {
  return `${(n ?? 0).toFixed(digits)}%`;
}

// formatUptime renders seconds as a compact "Nd Nh Nm" string.
export function formatUptime(seconds: number): string {
  if (!seconds || seconds < 0) return "—";
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

// countryToFlag converts a two-letter ISO country code to its emoji flag,
// avoiding a CDN dependency. Unknown codes render a globe.
export function countryToFlag(code: string): string {
  if (!code || code.length !== 2 || !/^[a-zA-Z]{2}$/.test(code)) return "\u{1F310}";
  const base = 0x1f1e6;
  const upper = code.toUpperCase();
  return String.fromCodePoint(
    base + (upper.charCodeAt(0) - 65),
    base + (upper.charCodeAt(1) - 65),
  );
}

// usageTone classifies a 0-100 usage value into a semantic tone.
export function usageTone(pct: number): "success" | "warning" | "critical" {
  if (pct >= 90) return "critical";
  if (pct >= 70) return "warning";
  return "success";
}
