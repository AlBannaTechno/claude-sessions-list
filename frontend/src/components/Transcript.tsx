import { useState, type MouseEvent, type ReactNode } from "react";
import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import { Browser } from "@wailsio/runtime";
import type { Entry } from "../api";
import { Button, FoldArrow } from "./ui";

/** How many entries show at first, and how many more each "Show earlier" adds. */
export const PAGE = 200;

// A link opens in the browser; followed inside the window it would replace the app.
const markdownParts: Components = {
  a: ({ href, children }) => (
    <a
      href={href}
      onClick={(e: MouseEvent) => {
        e.preventDefault();
        if (href && /^https?:/.test(href)) void Browser.OpenURL(href);
      }}
      title={href}
    >
      {children}
    </a>
  ),
};

export function Markdown({ text }: { text: string }) {
  return (
    <div className="markdown selectable">
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={markdownParts}>
        {text}
      </ReactMarkdown>
    </div>
  );
}

/** A session's conversation: the person's messages in a box, Claude's answers as markdown, each tool call as one line
 *  that opens to show what it was asked and what it answered. */
export function Transcript({ entries, showThinking }: { entries: Entry[]; showThinking: boolean }) {
  const shown = showThinking ? entries : entries.filter((e) => e.kind !== "thinking");
  const [limit, setLimit] = useState(PAGE);
  const start = Math.max(0, shown.length - limit);
  return (
    <div className="flex flex-col gap-2.5 max-w-[880px] mx-auto px-6 py-5">
      {start > 0 && (
        <div className="flex items-center gap-3">
          <Button small onClick={() => setLimit((n) => n + PAGE)}>Show {Math.min(PAGE, start)} earlier</Button>
          <span className="text-faint text-[12px]">{start} earlier entries not shown</span>
        </div>
      )}
      {shown.slice(start).map((e, i) => (
        <EntryView key={start + i} e={e} />
      ))}
    </div>
  );
}

function EntryView({ e }: { e: Entry }) {
  switch (e.kind) {
    case "user":
      return <div className="msg-user">{e.text}</div>;
    case "assistant":
      return <Markdown text={e.text} />;
    case "thinking":
      return (
        <div className="text-muted text-[12.5px] border-l-2 border-line-strong pl-3 italic whitespace-pre-wrap selectable">{e.text}</div>
      );
    case "summary":
      return <Folded label="Summary of the earlier conversation (made when the context was compacted)" body={<Markdown text={e.text} />} />;
    case "tool":
      return <ToolCall e={e} />;
    default:
      return null;
  }
}

function Folded({ label, body }: { label: string; body: ReactNode }) {
  const [open, setOpen] = useState(false);
  return (
    <div>
      <button className="tool-line" onClick={() => setOpen(!open)} aria-expanded={open}>
        <FoldArrow open={open} size={13} />
        <span className="truncate">{label}</span>
      </button>
      {open && <div className="tool-box !font-sans !text-[13px]">{body}</div>}
    </div>
  );
}

function ToolCall({ e }: { e: Entry }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="-my-1">
      <button className="tool-line" onClick={() => setOpen(!open)} aria-expanded={open}>
        <FoldArrow open={open} size={13} />
        <span className="font-medium text-ink-2 flex-none">{e.tool}</span>
        {e.brief && <span className="mono !text-[11.5px] truncate">{e.brief}</span>}
        {e.isError && <span className="flex-none text-[11.5px]" style={{ color: "var(--bad-fg)" }}>error</span>}
      </button>
      {open && (
        <div className="flex flex-col">
          <span className="label-caps !text-[10.5px] ml-5 mt-1">Input</span>
          <pre className="tool-box mono m-0">{e.input}</pre>
          <span className="label-caps !text-[10.5px] ml-5">{e.isError ? "Error" : "Result"}</span>
          <pre className="tool-box mono m-0" style={e.isError ? { color: "var(--bad-fg)" } : undefined}>{e.result || "(no answer recorded)"}</pre>
        </div>
      )}
    </div>
  );
}
