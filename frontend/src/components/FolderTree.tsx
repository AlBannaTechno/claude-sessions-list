import { useEffect, useRef, type KeyboardEvent } from "react";
import { RefreshCw, Search, X } from "lucide-react";
import type { Group, Summary } from "../api";
import { relAgo, shortId } from "../lib/format";
import { sources, visibleSessions, type Source } from "../lib/filter";
import { Chip, FoldArrow, Switch, iconProps } from "./ui";

type Props = {
  groups: Group[];
  totals: { sessions: number; folders: number } | null;
  selectedId: string | null;
  onSelect: (s: Summary) => void;
  isOpen: (key: string) => boolean;
  setOpen: (key: string, open: boolean) => void;
  query: string;
  setQuery: (q: string) => void;
  showArchived: boolean;
  setShowArchived: (v: boolean) => void;
  source: Source;
  setSource: (s: Source) => void;
  sourceCounts: Record<Source, number>;
  onRefresh: () => void;
  refreshing: boolean;
};

/** The left pane: every folder with its sessions under it, newest first, like the desktop app's sidebar used to be. */
export function FolderTree(p: Props) {
  const listRef = useRef<HTMLDivElement>(null);

  // The selected row stays in sight as the arrow keys move it.
  useEffect(() => {
    if (!p.selectedId) return;
    listRef.current?.querySelector(`[data-session="${CSS.escape(p.selectedId)}"]`)?.scrollIntoView({ block: "nearest" });
  }, [p.selectedId]);

  function onKey(e: KeyboardEvent) {
    const shown = visibleSessions(p.groups, p.isOpen);
    const at = shown.findIndex((s) => s.id === p.selectedId);
    const current = at >= 0 ? shown[at] : null;
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      const next = shown[e.key === "ArrowDown" ? Math.min(at + 1, shown.length - 1) : Math.max(at - 1, 0)];
      if (next) p.onSelect(next);
    } else if ((e.key === "ArrowLeft" || e.key === "ArrowRight") && current) {
      e.preventDefault();
      p.setOpen(current.groupKey, e.key === "ArrowRight");
    }
  }

  const visible = p.groups.reduce((n, g) => n + (g.sessions?.length ?? 0), 0);

  return (
    <aside className="w-[320px] flex-none h-full flex flex-col bg-sidebar border-r border-line">
      <div className="h-11 flex-none flex items-center justify-end px-2.5 [--wails-draggable:drag]">
        <button
          className="flex items-center justify-center w-[26px] h-[24px] rounded-[6px] text-ink-2 hover:bg-hover [--wails-draggable:no-drag]"
          onClick={p.onRefresh}
          disabled={p.refreshing}
          title="Read the sessions again"
          aria-label="Read the sessions again"
        >
          <RefreshCw {...iconProps} className={p.refreshing ? "animate-spin" : ""} />
        </button>
      </div>
      <div className="px-2.5 flex flex-col gap-2 pb-2">
        <div className="field">
          <Search size={14} strokeWidth={1.75} className="text-faint flex-none" />
          <input value={p.query} onChange={(e) => p.setQuery(e.target.value)} placeholder="Search name, id, folder or prompt" aria-label="Search sessions" spellCheck={false} />
          {p.query && (
            <button className="text-faint" aria-label="Clear search" onClick={() => p.setQuery("")}>
              <X size={13} />
            </button>
          )}
        </div>
        <SourceFilter source={p.source} setSource={p.setSource} counts={p.sourceCounts} />
        <div className="flex items-center justify-between">
          <Switch checked={p.showArchived} onChange={p.setShowArchived} label="Show archived" />
          {p.totals && (
            <span className="text-faint text-[12px]" title={`${p.totals.sessions} sessions in ${p.totals.folders} folders on this computer`}>
              {visible === p.totals.sessions ? `${visible} sessions` : `${visible} of ${p.totals.sessions}`} · {p.groups.length} folders
            </span>
          )}
        </div>
      </div>
      <div className="hr mx-2.5" />
      <div ref={listRef} className="scroll flex-1 px-2.5 py-2 outline-none" tabIndex={0} onKeyDown={onKey} aria-label="Sessions by folder">
        {p.groups.map((g) => {
          const open = p.isOpen(g.key);
          const sessions = g.sessions ?? [];
          return (
            <div key={g.key} className="mb-0.5">
              <button className="folder-row" onClick={() => p.setOpen(g.key, !open)} title={g.path || g.label} aria-expanded={open}>
                <FoldArrow open={open} />
                <span className="truncate">{g.label}</span>
                <span className="ml-auto text-faint text-[12px] font-normal tabular-nums">{sessions.length}</span>
              </button>
              {open && (
                <div className="flex flex-col gap-px pb-1">
                  {sessions.map((s) => (
                    <SessionRow key={s.id} s={s} selected={s.id === p.selectedId} onSelect={() => p.onSelect(s)} />
                  ))}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </aside>
  );
}

/** All, CLI or Desktop: which sessions the list shows, by where they were started. Each shows how many it has. */
export function SourceFilter({ source, setSource, counts }: { source: Source; setSource: (s: Source) => void; counts: Record<Source, number> }) {
  return (
    <div className="seg w-full" role="group" aria-label="Which sessions to list">
      {sources.map((o) => (
        <button key={o.value} className="flex-1 flex items-center justify-center gap-1.5" aria-pressed={source === o.value} onClick={() => setSource(o.value)} title={o.title}>
          {o.label}
          <span className="text-faint text-[11.5px] font-normal tabular-nums">{counts[o.value]}</span>
        </button>
      ))}
    </div>
  );
}

function SessionRow({ s, selected, onSelect }: { s: Summary; selected: boolean; onSelect: () => void }) {
  return (
    <button className="session-row" aria-current={selected ? "page" : undefined} onClick={onSelect} data-session={s.id} title={s.firstPrompt || s.title}>
      <span className="session-name truncate">{s.title}</span>
      <span className="flex items-center gap-1.5 min-w-0 text-faint text-[12px]">
        <span className="mono !text-[11.5px]">{shortId(s.id)}</span>
        <span>·</span>
        <span className="truncate">{relAgo(s.lastActive)}</span>
        <span className="flex-1" />
        {s.worktree && <Chip title={`Ran in the worktree ${s.worktree}`}>{s.worktree}</Chip>}
        {s.archived && <Chip title="Archived in the Claude app">archived</Chip>}
        {!s.localId && <Chip title="Started from the terminal, not the Claude app">CLI</Chip>}
      </span>
    </button>
  );
}
