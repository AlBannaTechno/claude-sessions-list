import type { Group, Summary } from "../api";

/** Whether a session matches every word typed in search: its name, ids, folder, worktree or first prompt. */
export function matches(s: Summary, groupLabel: string, words: string[]): boolean {
  if (!words.length) return true;
  const hay = [s.title, s.id, s.localId, s.folder, s.worktree, s.gitBranch, s.firstPrompt, groupLabel].join("\n").toLowerCase();
  return words.every((w) => hay.includes(w));
}

/** Which sessions to list by where they were started: the Claude desktop app, the terminal (CLI), or both. */
export type Source = "all" | "cli" | "desktop";

export const sources: { value: Source; label: string; title: string }[] = [
  { value: "all", label: "All", title: "Sessions from the Claude desktop app and from the terminal" },
  { value: "cli", label: "CLI", title: "Only sessions started from the terminal with the claude command" },
  { value: "desktop", label: "Desktop", title: "Only sessions started in the Claude desktop app" },
];

/** Most people use the desktop app, so its sessions are what the list shows until they pick another. */
export const defaultSource: Source = "desktop";

/** A session the desktop app knows has its desktop id (local_…); one started in the terminal has none. */
export function fromSource(s: Summary, source: Source): boolean {
  if (source === "desktop") return !!s.localId;
  if (source === "cli") return !s.localId;
  return true;
}

export type Filters = { query: string; showArchived: boolean; source: Source };

/** The groups to show: sessions from the chosen source, archived ones only when asked, then only those matching the
 *  search. Groups left empty go. */
export function filterGroups(groups: Group[], { query, showArchived, source }: Filters): Group[] {
  const words = query.toLowerCase().split(/\s+/).filter(Boolean);
  const out: Group[] = [];
  for (const g of groups) {
    const sessions = (g.sessions ?? []).filter((s) => fromSource(s, source) && (showArchived || !s.archived) && matches(s, g.label, words));
    if (sessions.length) out.push({ ...g, sessions });
  }
  return out;
}

/** The sessions in the order they are drawn: only those inside open groups. Arrow keys move along this list. */
export function visibleSessions(groups: Group[], isOpen: (key: string) => boolean): Summary[] {
  return groups.flatMap((g) => (isOpen(g.key) ? g.sessions ?? [] : []));
}
