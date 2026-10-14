import { describe, expect, it } from "vitest";
import type { Group, Summary } from "../api";
import { filterGroups, visibleSessions } from "./filter";
import { hasTime, relAgo } from "./format";

function session(over: Partial<Summary>): Summary {
  return {
    id: "00000000-0000-4000-8000-000000000001", localId: "", title: "", titleFrom: "desktop", folder: "", groupKey: "", worktree: "",
    gitBranch: "", model: "", archived: false, hasTranscript: true, started: "", lastActive: "", prompts: 1, firstPrompt: "", ...over,
  };
}

const groups: Group[] = [
  {
    key: "/u/notes-app", label: "notes-app", path: "/u/notes-app", lastActive: "",
    sessions: [
      session({ id: "a1", title: "Fix the login page", groupKey: "/u/notes-app" }),
      session({ id: "a2", title: "Old work", archived: true, groupKey: "/u/notes-app" }),
    ],
  },
  { key: "/u/shop", label: "shop", path: "/u/shop", lastActive: "", sessions: [session({ id: "b1", title: "Add a cart", firstPrompt: "please add a cart", groupKey: "/u/shop" })] },
];

describe("filterGroups", () => {
  it("hides archived sessions unless asked", () => {
    expect(filterGroups(groups, "", false)[0].sessions!.map((s) => s.id)).toEqual(["a1"]);
    expect(filterGroups(groups, "", true)[0].sessions!.map((s) => s.id)).toEqual(["a1", "a2"]);
  });

  it("needs every word, in any field, and drops folders left empty", () => {
    expect(filterGroups(groups, "login PAGE", false).map((g) => g.label)).toEqual(["notes-app"]);
    expect(filterGroups(groups, "please cart", false).map((g) => g.label)).toEqual(["shop"]);
    expect(filterGroups(groups, "SHOP", false)[0].sessions).toHaveLength(1); // the folder's name counts too
    expect(filterGroups(groups, "b1", false).map((g) => g.label)).toEqual(["shop"]);
    expect(filterGroups(groups, "nothing like this", false)).toEqual([]);
  });

  it("walks only the sessions of open folders", () => {
    expect(visibleSessions(groups, (k) => k === "/u/shop").map((s) => s.id)).toEqual(["b1"]);
  });
});

describe("times", () => {
  it("treats Go's empty time as no time", () => {
    expect(hasTime("0001-01-01T00:00:00Z")).toBe(false);
    expect(relAgo("0001-01-01T00:00:00Z")).toBe("");
    expect(relAgo("2026-10-14T10:00:00Z", new Date("2026-10-14T10:05:00Z"))).toBe("5 min ago");
  });
});
