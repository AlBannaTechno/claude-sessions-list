// Copied from PR Watcher's frontend/src/lib/format.ts.

function sameDay(a: Date, b: Date) {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

/** Go sends a time it does not know as year 1; that is no time at all. */
export function hasTime(iso: string | null | undefined): iso is string {
  return !!iso && !iso.startsWith("0001-");
}

/** "now", "6 min", "2 h", "Yesterday", "8 Sep". */
export function relTime(iso: string | null | undefined, now = new Date()): string {
  if (!hasTime(iso)) return "";
  const t = new Date(iso);
  const sec = Math.max(0, (now.getTime() - t.getTime()) / 1000);
  if (sec < 60) return "now";
  if (sec < 3600) return `${Math.floor(sec / 60)} min`;
  if (sameDay(t, now)) return `${Math.floor(sec / 3600)} h`;
  const y = new Date(now);
  y.setDate(y.getDate() - 1);
  if (sameDay(t, y)) return "Yesterday";
  if (t.getFullYear() !== now.getFullYear()) return t.toLocaleDateString(undefined, { day: "numeric", month: "short", year: "numeric" });
  return t.toLocaleDateString(undefined, { day: "numeric", month: "short" });
}

/** relTime after a verb: "just now", "6 min ago", "2 h ago", "yesterday", "on 8 Sep". */
export function relAgo(iso: string | null | undefined, now = new Date()): string {
  const r = relTime(iso, now);
  if (!r) return "";
  if (r === "now") return "just now";
  if (r === "Yesterday") return "yesterday";
  if (/^\d+ (min|h)$/.test(r)) return `${r} ago`;
  return `on ${r}`;
}

/** A full date and time, for the line under a session's name. */
export function dateTime(iso: string | null | undefined): string {
  if (!hasTime(iso)) return "";
  return new Date(iso).toLocaleString(undefined, { day: "numeric", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit", hour12: false });
}

export function errorText(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === "string") return e;
  if (e && typeof e === "object" && "message" in e) return String((e as { message: unknown }).message);
  return "Something went wrong";
}

/** The first part of an id, enough to tell sessions apart in a list. */
export function shortId(id: string): string {
  return id.replace(/^local_/, "").slice(0, 8);
}
