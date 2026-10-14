import { useCallback, useEffect, useMemo, useState } from "react";
import { useListing, useRefreshOnFocus, type Summary } from "./api";
import { filterGroups } from "./lib/filter";
import { FolderTree } from "./components/FolderTree";
import { SessionView } from "./components/SessionView";
import { Empty, LoadFailed, Loading } from "./components/ui";

// What the window remembers between runs: the open folders and the session last shown. Browser storage can be
// missing or refuse, so every read and write is guarded and the app works without it.
function load<T>(key: string, fallback: T): T {
  try {
    const v = localStorage.getItem(key);
    return v ? (JSON.parse(v) as T) : fallback;
  } catch {
    return fallback;
  }
}
function save(key: string, v: unknown) {
  try {
    localStorage.setItem(key, JSON.stringify(v));
  } catch {
    /* not kept; nothing else changes */
  }
}

export default function App() {
  useRefreshOnFocus();
  const listing = useListing();
  const [query, setQuery] = useState("");
  const [showArchived, setShowArchived] = useState(() => load("showArchived", false));
  const [openKeys, setOpenKeys] = useState<string[]>(() => load("openFolders", []));
  const [selectedId, setSelectedId] = useState<string | null>(() => load("selected", null));

  useEffect(() => save("showArchived", showArchived), [showArchived]);
  useEffect(() => save("openFolders", openKeys), [openKeys]);
  useEffect(() => save("selected", selectedId), [selectedId]);

  const all = listing.data?.groups ?? [];
  const groups = useMemo(() => filterGroups(all, query, showArchived), [all, query, showArchived]);
  const selected: Summary | null = useMemo(() => {
    for (const g of all) for (const s of g.sessions ?? []) if (s.id === selectedId) return s;
    return null;
  }, [all, selectedId]);

  // While searching every folder with a match is open, so the matches show without clicking.
  const isOpen = useCallback((key: string) => !!query.trim() || openKeys.includes(key), [query, openKeys]);
  const setOpen = useCallback((key: string, open: boolean) => {
    setOpenKeys((ks) => (open ? (ks.includes(key) ? ks : [...ks, key]) : ks.filter((k) => k !== key)));
  }, []);
  const select = useCallback((s: Summary) => {
    setSelectedId(s.id);
    setOpenKeys((ks) => (ks.includes(s.groupKey) ? ks : [...ks, s.groupKey]));
  }, []);

  return (
    <div className="flex h-full">
      <FolderTree
        groups={groups}
        totals={listing.data ? { sessions: listing.data.sessions, folders: all.length } : null}
        selectedId={selectedId}
        onSelect={select}
        isOpen={isOpen}
        setOpen={setOpen}
        query={query}
        setQuery={setQuery}
        showArchived={showArchived}
        setShowArchived={setShowArchived}
        onRefresh={() => void listing.refetch()}
        refreshing={listing.isFetching}
      />
      {listing.isError ? (
        <main className="flex-1 min-w-0">
          <div className="h-11 [--wails-draggable:drag]" />
          <LoadFailed what="the sessions" error={listing.error} retry={() => void listing.refetch()} />
        </main>
      ) : !listing.data ? (
        <main className="flex-1 min-w-0 px-6 pt-14">
          <Loading>Reading the sessions in ~/.claude/projects…</Loading>
        </main>
      ) : selected ? (
        <SessionView key={selected.id} s={selected} />
      ) : (
        <main className="flex-1 min-w-0">
          <div className="h-11 [--wails-draggable:drag]" />
          <Empty
            title={groups.length ? "Pick a session" : "No sessions match"}
            body={
              groups.length
                ? "Open a folder on the left and click a session to read its conversation."
                : "Nothing matches the search. Clear it, or turn on Show archived."
            }
          />
        </main>
      )}
    </div>
  );
}
