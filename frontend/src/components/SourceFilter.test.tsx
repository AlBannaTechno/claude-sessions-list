import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { SourceFilter } from "./FolderTree";

describe("SourceFilter", () => {
  it("offers All, CLI and Desktop in that order, with how many each has, and marks the one in use", () => {
    render(<SourceFilter source="desktop" setSource={() => {}} counts={{ all: 678, cli: 240, desktop: 438 }} />);
    const buttons = screen.getAllByRole("button");
    expect(buttons.map((b) => b.textContent)).toEqual(["All678", "CLI240", "Desktop438"]);
    expect(screen.getByRole("button", { name: /Desktop/ })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: /All/ })).toHaveAttribute("aria-pressed", "false");
  });

  it("switches to the one clicked", () => {
    const setSource = vi.fn();
    render(<SourceFilter source="desktop" setSource={setSource} counts={{ all: 3, cli: 1, desktop: 2 }} />);
    fireEvent.click(screen.getByRole("button", { name: /CLI/ }));
    expect(setSource).toHaveBeenCalledWith("cli");
  });
});
