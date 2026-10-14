// The pieces used here, copied from PR Watcher's frontend/src/components/ui.tsx.
import { createContext, useCallback, useContext, useState, type ButtonHTMLAttributes, type ReactNode } from "react";
import * as RSwitch from "@radix-ui/react-switch";
import { ChevronRight, TriangleAlert, X } from "lucide-react";
import { errorText } from "../lib/format";

export const iconProps = { size: 15, strokeWidth: 1.75 } as const;

/** A chip of words. */
export function Chip({ tint, title, children }: { tint?: "you" | "ok" | "bad"; title?: string; children: ReactNode }) {
  return (
    <span className={`${tint ? `chip chip-${tint}` : "chip"} flex-none`} title={title}>
      {children}
    </span>
  );
}

/** The arrow of a row that opens on click: it points down while open. */
export function FoldArrow({ open, size = 14 }: { open: boolean; size?: number }) {
  return <ChevronRight size={size} className="flex-none text-faint" style={{ transform: open ? "rotate(90deg)" : "none" }} aria-hidden data-open={open} />;
}

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & { variant?: "primary" | "danger"; small?: boolean; icon?: ReactNode };

export function Button({ variant, small, icon, className = "", children, ...rest }: ButtonProps) {
  const cls = ["btn", variant === "primary" && "btn-primary", variant === "danger" && "btn-danger", small && "btn-small", className].filter(Boolean).join(" ");
  return (
    <button type="button" className={cls} {...rest}>
      {icon}
      {children}
    </button>
  );
}

/** A switch always carries a visible label next to it. */
export function Switch({ checked, onChange, label }: { checked: boolean; onChange: (v: boolean) => void; label: string }) {
  return (
    <label className="flex items-center gap-2 text-[12px] text-muted cursor-default">
      <RSwitch.Root className="switch" checked={checked} onCheckedChange={onChange} aria-label={label}>
        <RSwitch.Thumb className="switch-thumb" />
      </RSwitch.Root>
      {label}
    </label>
  );
}

export function Header({ title, sub, right }: { title: string; sub?: ReactNode; right?: ReactNode }) {
  return (
    <div className="h-14 flex-none flex items-center gap-3 px-6 border-b border-line [--wails-draggable:drag]">
      <div className="flex items-baseline gap-2.5 min-w-0">
        <span className="text-[15px] font-semibold truncate" title={title}>{title}</span>
        {sub && <span className="text-faint text-[12.5px] truncate">{sub}</span>}
      </div>
      <div className="flex-1" />
      <div className="flex items-center gap-2 flex-none [--wails-draggable:no-drag]">{right}</div>
    </div>
  );
}

export function Banner({ title, body, action }: { title: string; body: ReactNode; action?: ReactNode }) {
  return (
    <div className="flex gap-3 items-start rounded-lg px-3.5 py-3" style={{ border: "1px solid var(--warn-border)", background: "var(--warn-bg)" }}>
      <TriangleAlert size={16} strokeWidth={1.75} style={{ color: "var(--need-dot)", flex: "none", marginTop: 1 }} />
      <div className="flex-1 flex flex-col gap-0.5 min-w-0">
        <span className="font-semibold" style={{ color: "var(--warn-title)" }}>{title}</span>
        <span className="leading-[1.45] selectable" style={{ color: "var(--warn-text)" }}>{body}</span>
      </div>
      {action}
    </div>
  );
}

/** A read that failed: it says so, with the reason and Retry, never "nothing here". */
export function LoadFailed({ what, error, retry }: { what: string; error: unknown; retry?: () => void }) {
  return (
    <div className="px-6 pt-4" role="alert">
      <Banner title={`Could not load ${what}`} body={errorText(error)} action={retry && <Button small onClick={() => retry()}>Retry</Button>} />
    </div>
  );
}

/** Data still on its way: one muted line in plain words. */
export function Loading({ children }: { children: ReactNode }) {
  return <p className="text-muted text-[12.5px] m-0 py-2" role="status">{children}</p>;
}

export function Empty({ title, body, action }: { title: string; body?: ReactNode; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-start gap-2 px-6 py-10 max-w-[520px]">
      <span className="text-[14px] font-semibold">{title}</span>
      {body && <span className="text-muted leading-[1.5]">{body}</span>}
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
}

// ---- messages after actions ----

type Toast = { id: number; text: string; error: boolean };
const ToastCtx = createContext<(text: string, error?: boolean) => void>(() => {});

export function ToastHost({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const show = useCallback((text: string, error = false) => {
    const id = Date.now() + Math.random();
    setToasts((t) => [...t, { id, text, error }]);
    setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), error ? 8000 : 3500);
  }, []);
  return (
    <ToastCtx.Provider value={show}>
      {children}
      <div className="fixed bottom-4 right-4 flex flex-col gap-2 z-50" aria-live="polite">
        {toasts.map((t) => (
          <div key={t.id} className="pop flex items-start gap-2 px-3 py-2.5 max-w-[420px]" role={t.error ? "alert" : "status"}>
            {t.error && <span className="dot mt-1.5" style={{ background: "var(--bad-dot)" }} />}
            <span className="flex-1 leading-[1.45] selectable">{t.text}</span>
            <button aria-label="Close message" className="text-faint" onClick={() => setToasts((x) => x.filter((y) => y.id !== t.id))}>
              <X size={14} />
            </button>
          </div>
        ))}
      </div>
    </ToastCtx.Provider>
  );
}

export const useToast = () => useContext(ToastCtx);

/** Runs an action and shows its error, or the done message, as a toast. */
export function useAction<A extends unknown[]>(fn: (...a: A) => Promise<unknown>, done?: string | ((r: unknown) => string)) {
  const toast = useToast();
  return useCallback(
    async (...a: A) => {
      try {
        const r = await fn(...a);
        if (done) toast(typeof done === "string" ? done : done(r));
      } catch (e) {
        toast(errorText(e), true);
      }
    },
    [fn, done, toast],
  );
}
