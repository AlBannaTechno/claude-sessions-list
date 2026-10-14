import { useState } from "react";
import { Copy, FolderOpen, SquareArrowOutUpRight, Terminal } from "lucide-react";
import { Service, useTranscript, type Summary } from "../api";
import { dateTime, relAgo } from "../lib/format";
import { Button, Empty, Header, LoadFailed, Loading, Switch, iconProps, useAction } from "./ui";
import { Transcript } from "./Transcript";

/** The right pane: one session's name, where it ran, buttons to open it elsewhere, and its conversation. */
export function SessionView({ s }: { s: Summary }) {
  const transcript = useTranscript(s.id, s.hasTranscript);
  const [showThinking, setShowThinking] = useState(false);

  const openInClaude = useAction(() => Service.OpenInDesktop(s.localId));
  const copyResume = useAction(() => Service.CopyResumeCommand(s.folder, s.id), (cmd) => `Copied: ${cmd}`);
  const copyId = useAction(() => Service.CopyText(s.id), "Session id copied");
  const reveal = useAction(() => Service.RevealFolder(s.folder));

  const facts = [
    s.model,
    s.gitBranch && `branch ${s.gitBranch}`,
    s.worktree && `worktree ${s.worktree}`,
    s.prompts > 0 && `${s.prompts} prompt${s.prompts === 1 ? "" : "s"}`,
    dateTime(s.started) && `started ${dateTime(s.started)}`,
    relAgo(s.lastActive) && `last active ${relAgo(s.lastActive)}`,
    s.archived && "archived",
  ].filter(Boolean);

  return (
    <main className="flex-1 min-w-0 flex flex-col h-full">
      <Header
        title={s.title}
        right={
          <>
            {s.localId && (
              <Button icon={<SquareArrowOutUpRight {...iconProps} />} onClick={() => void openInClaude()} title="Open this session in the Claude desktop app">
                Open in Claude
              </Button>
            )}
            <Button icon={<Terminal {...iconProps} />} onClick={() => void copyResume()} title="Copy the command that continues this session in the terminal">
              Copy resume command
            </Button>
            {s.folder && (
              <Button icon={<FolderOpen {...iconProps} />} onClick={() => void reveal()} title={`Open ${s.folder} in Finder (Mac) or File Explorer (Windows)`}>
                Show folder
              </Button>
            )}
          </>
        }
      />
      <div className="flex-none px-6 py-2.5 border-b border-line flex flex-col gap-1 text-[12.5px]">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-muted flex-none w-[52px]">Folder</span>
          <span className="selectable truncate" title={s.folder}>{s.folder || "Not recorded"}</span>
        </div>
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-muted flex-none w-[52px]">Id</span>
          <span className="mono selectable truncate">{s.id}</span>
          {s.localId && <span className="mono selectable text-faint truncate" title="The Claude app's id for this session">{s.localId}</span>}
          <button className="text-faint hover:text-ink flex-none" onClick={() => void copyId()} title="Copy the session id" aria-label="Copy the session id">
            <Copy size={13} strokeWidth={1.75} />
          </button>
        </div>
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-faint truncate flex-1">{facts.join(" · ")}</span>
          <Switch checked={showThinking} onChange={setShowThinking} label="Show thinking" />
        </div>
      </div>
      <div className="scroll flex-1">
        {!s.hasTranscript ? (
          <Empty title="No conversation on this computer" body="The Claude app lists this session, but its transcript file is not in ~/.claude/projects. It may have run in the cloud or on another computer." />
        ) : transcript.isError ? (
          <LoadFailed what="this conversation" error={transcript.error} retry={() => void transcript.refetch()} />
        ) : !transcript.data ? (
          <div className="px-6 py-4">
            <Loading>Reading the conversation…</Loading>
          </div>
        ) : transcript.data.entries?.length ? (
          <Transcript key={s.id} entries={transcript.data.entries} showThinking={showThinking} />
        ) : (
          <Empty title="Nothing to show" body="This transcript has no messages yet." />
        )}
      </div>
    </main>
  );
}
