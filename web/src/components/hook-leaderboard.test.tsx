import { describe, it, expect } from "vitest";
import { render, screen, within } from "@testing-library/react";
import { HookLeaderboard } from "./hook-leaderboard";
import type { HookStat } from "@/lib/types";

describe("HookLeaderboard", () => {
  it("renders empty states when there are no hook stats", () => {
    render(<HookLeaderboard hookStats={[]} />);

    expect(screen.getByText("No hook durations recorded yet.")).toBeInTheDocument();
    expect(screen.getByText("No failures recorded yet — nice!")).toBeInTheDocument();
  });

  it("ranks the slowest hooks by average duration, capped at 5", () => {
    const stats: HookStat[] = Array.from({ length: 7 }, (_, i) => ({
      name: `Hook ${i}`,
      total_runs: 10,
      total_failures: 0,
      failure_rate: 0,
      avg_duration_ms: (i + 1) * 100,
    }));
    render(<HookLeaderboard hookStats={stats} />);

    // Slowest (Hook 6, 700ms) should be listed; the 6th/7th slowest are cut.
    expect(screen.getByText("Hook 6")).toBeInTheDocument();
    expect(screen.queryByText("Hook 0")).not.toBeInTheDocument();
  });

  it("only lists hooks with at least one failure under flakiest, ranked by rate", () => {
    const stats: HookStat[] = [
      { name: "Reliable", total_runs: 10, total_failures: 0, failure_rate: 0, avg_duration_ms: 10 },
      { name: "Flaky", total_runs: 10, total_failures: 3, failure_rate: 30, avg_duration_ms: 10 },
      { name: "Flakier", total_runs: 10, total_failures: 8, failure_rate: 80, avg_duration_ms: 10 },
    ];
    render(<HookLeaderboard hookStats={stats} />);

    // "Flaky" legitimately appears in both lists here (same avg duration as
    // "Reliable" puts it in "slowest" too); what matters is that "Reliable"
    // (0 failures) never appears specifically under "Flakiest hooks".
    expect(screen.getAllByText("Flaky").length).toBeGreaterThan(0);
    expect(screen.getByText("30% (3/10)")).toBeInTheDocument();
    const flakiestCard = screen.getByText("🎲 Flakiest hooks").closest('[data-slot="card"]') as HTMLElement;
    expect(flakiestCard).not.toBeNull();
    expect(within(flakiestCard).queryByText("Reliable")).not.toBeInTheDocument();
  });
});
