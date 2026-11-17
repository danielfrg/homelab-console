// Formatters are built once: these run per table row and the locale can't change mid-page.
let timestampFormat: Intl.DateTimeFormat | undefined;

/** An ISO timestamp as "42s ago", "5m ago", "3h ago" or "2d ago"; a dash when empty. */
export function formatRelativeTime(value: string | null | undefined, now = Date.now()): string {
  if (!value) return "—";
  const seconds = Math.max(0, Math.round((now - Date.parse(value)) / 1000));
  if (seconds < 60) return `${seconds}s ago`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.round(seconds / 3600)}h ago`;
  return `${Math.round(seconds / 86400)}d ago`;
}

/** An ISO timestamp as a short date and time; a dash when empty. */
export function formatTimestamp(value: string | null | undefined): string {
  if (!value) return "—";
  timestampFormat ??= new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
  return timestampFormat.format(new Date(value));
}
