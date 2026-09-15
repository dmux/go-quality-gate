import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { AchievementCard } from "./achievement-card";
import type { Achievement } from "@/lib/types";

describe("AchievementCard", () => {
  it("renders an unlocked achievement with the Unlocked badge and no progress bar", () => {
    const achievement: Achievement = {
      id: "first_steps",
      name: "First Steps",
      description: "Ran quality-gate for the first time",
      unlocked: true,
      progress: 1,
      target: 1,
    };
    const { container } = render(<AchievementCard achievement={achievement} />);

    expect(screen.getByText(/First Steps/)).toBeInTheDocument();
    expect(screen.getByText("Unlocked")).toBeInTheDocument();
    expect(container.querySelector('[role="progressbar"]')).toBeNull();
  });

  it("renders a locked achievement with a progress fraction and a progress bar", () => {
    const achievement: Achievement = {
      id: "clean_coder",
      name: "Clean Coder",
      description: "Auto-fixed 50 issues with --fix",
      unlocked: false,
      progress: 12,
      target: 50,
    };
    const { container } = render(<AchievementCard achievement={achievement} />);

    expect(screen.getByText("12/50")).toBeInTheDocument();
    expect(container.querySelector('[role="progressbar"]')).not.toBeNull();
  });

  it("handles a zero target without dividing by zero", () => {
    const achievement: Achievement = {
      id: "weird",
      name: "Weird",
      description: "Edge case",
      unlocked: false,
      progress: 0,
      target: 0,
    };
    render(<AchievementCard achievement={achievement} />);

    expect(screen.getByText("0/0")).toBeInTheDocument();
  });
});
