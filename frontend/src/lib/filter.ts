import type { Group, Summary } from "../api";

/** Whether a session matches every word typed in search: its name, ids, folder, worktree or first prompt. */
export function matches(s: Summary, groupLabel: string, words: string[]): boolean {
  if (!words.length) return true;
  const hay = [s.title, s.id, s.localId, s.folder, s.worktree, s.gitBranch, s.firstPrompt, groupLabel].join("\n").toLowerCase();
  return words.every((w) => hay.includes(w));
}

/** The groups to show: archived sessions only when asked, then only sessions matching the search. Empty groups go. */
export function filterGroups(groups: Group[], query: string, showArchived: boolean): Group[] {
  const words = query.toLowerCase().split(/\s+/).filter(Boolean);
  const out: Group[] = [];
  for (const g of groups) {
    const sessions = (g.sessions ?? []).filter((s) => (showArchived || !s.archived) && matches(s, g.label, words));
    if (sessions.length) out.push({ ...g, sessions });
  }
  return out;
}

/** The sessions in the order they are drawn: only those inside open groups. Arrow keys move along this list. */
export function visibleSessions(groups: Group[], isOpen: (key: string) => boolean): Summary[] {
  return groups.flatMap((g) => (isOpen(g.key) ? g.sessions ?? [] : []));
}
