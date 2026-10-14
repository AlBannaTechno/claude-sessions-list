import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import type { Entry } from "../api";
import { PAGE, Transcript } from "./Transcript";

vi.mock("@wailsio/runtime", () => ({ Browser: { OpenURL: vi.fn() } }));

function entry(over: Partial<Entry>): Entry {
  return { kind: "assistant", text: "", tool: "", brief: "", input: "", result: "", isError: false, time: "", ...over };
}

describe("Transcript", () => {
  it("draws markdown tables from Claude's answers", () => {
    render(<Transcript showThinking={false} entries={[entry({ text: "| a | b |\n|---|---|\n| 1 | 2 |" })]} />);
    expect(screen.getByRole("table")).toBeInTheDocument();
    expect(screen.getByRole("cell", { name: "2" })).toBeInTheDocument();
  });

  it("shows a tool call as one line that opens to its input and answer", () => {
    render(
      <Transcript showThinking={false} entries={[entry({ kind: "tool", tool: "Bash", brief: "git status", input: '{"command":"git status"}', result: "clean", isError: false })]} />,
    );
    const line = screen.getByRole("button", { name: /Bash/ });
    expect(line).toHaveTextContent("git status");
    expect(screen.queryByText("clean")).not.toBeInTheDocument();
    fireEvent.click(line);
    expect(screen.getByText("clean")).toBeInTheDocument();
    expect(line).toHaveAttribute("aria-expanded", "true");
  });

  it("hides thinking unless asked", () => {
    const entries = [entry({ kind: "thinking", text: "private reasoning" }), entry({ kind: "user", text: "hello" })];
    const { rerender } = render(<Transcript showThinking={false} entries={entries} />);
    expect(screen.queryByText("private reasoning")).not.toBeInTheDocument();
    expect(screen.getByText("hello")).toBeInTheDocument();
    rerender(<Transcript showThinking entries={entries} />);
    expect(screen.getByText("private reasoning")).toBeInTheDocument();
  });

  it("shows the newest entries first and the earlier ones on request", () => {
    const entries = Array.from({ length: PAGE + 5 }, (_, i) => entry({ kind: "user", text: `message ${i}` }));
    render(<Transcript showThinking={false} entries={entries} />);
    expect(screen.queryByText("message 0")).not.toBeInTheDocument();
    expect(screen.getByText(`message ${PAGE + 4}`)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Show 5 earlier" }));
    expect(screen.getByText("message 0")).toBeInTheDocument();
  });
});
