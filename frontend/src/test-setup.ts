import "@testing-library/jest-dom/vitest";
import { afterEach } from "vitest";
import { cleanup } from "@testing-library/react";

/** Every button inside another button on the page: invalid HTML, and a click on the inner one opens the row too. */
export function nestedButtons(root: ParentNode = document.body): string[] {
  const name = (el: Element) => (el.getAttribute("aria-label") ?? el.textContent ?? "").trim().slice(0, 60);
  return [...root.querySelectorAll("button button, button [role=button]")].map((inner) => `"${name(inner)}" inside "${name(inner.parentElement!.closest("button")!)}"`);
}

// Copied from PR Watcher: each test checks that no button sits inside another, then clears the page.
afterEach(() => {
  const nested = nestedButtons();
  cleanup();
  if (nested.length) throw new Error(`A button inside a button: ${nested.join("; ")}`);
});

// jsdom has no scrolling.
Element.prototype.scrollIntoView = function () {};
