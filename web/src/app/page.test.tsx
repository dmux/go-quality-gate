import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import Home from "./page";
import type { StatsResponse } from "@/lib/types";

const fullStats: StatsResponse = {
  stats: {
    total_runs: 4,
    total_failed_runs: 1,
    success_rate: 75,
    streak_days: 5,
    streak_hook_type: "pre-commit",
    streak_broken_today: false,
    total_auto_fixes: 2,
    fixes_this_month: 1,
    time_saved_this_month_minutes: 5,
  },
  achievements: [
    { id: "first_steps", name: "First Steps", description: "d", unlocked: true, progress: 1, target: 1 },
  ],
  daily: [{ date: "2026-09-13", total: 1, failed: 0, success_rate: 100 }],
  recent_runs: [
    {
      timestamp: "2026-09-13T11:37:32-03:00",
      hook_type: "pre-commit",
      success: true,
      duration_ms: 10,
      hooks: [{ name: "Say hi", success: true, duration_ms: 10 }],
    },
  ],
  hook_stats: [{ name: "Say hi", total_runs: 4, total_failures: 1, failure_rate: 25, avg_duration_ms: 10 }],
};

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Home page", () => {
  it("shows a loading state before the fetch resolves", () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(() => new Promise(() => {})) // never resolves
    );

    render(<Home />);

    expect(screen.getByText("Loading…")).toBeInTheDocument();
  });

  it("shows an error state when the fetch response is not ok", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(() => Promise.resolve({ ok: false, status: 500 } as Response))
    );

    render(<Home />);

    await waitFor(() => {
      expect(screen.getByText(/Failed to load data/)).toBeInTheDocument();
    });
  });

  it("shows an error state when the fetch itself rejects", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(() => Promise.reject(new Error("network down")))
    );

    render(<Home />);

    await waitFor(() => {
      expect(screen.getByText(/Failed to load data/)).toBeInTheDocument();
    });
  });

  it("renders the full dashboard once data loads", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        Promise.resolve({
          ok: true,
          json: () => Promise.resolve(fullStats),
        } as Response)
      )
    );

    render(<Home />);

    await waitFor(() => {
      expect(screen.getByText("Quality Gate Dashboard")).toBeInTheDocument();
    });

    expect(screen.getByText("5 days")).toBeInTheDocument();
    expect(screen.getByText("75%")).toBeInTheDocument();
    expect(screen.getByText("First Steps", { exact: false })).toBeInTheDocument();
  });

  it("shows the streak-broken hint when streak_broken_today is true", async () => {
    const broken: StatsResponse = {
      ...fullStats,
      stats: { ...fullStats.stats, streak_broken_today: true },
    };
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        Promise.resolve({
          ok: true,
          json: () => Promise.resolve(broken),
        } as Response)
      )
    );

    render(<Home />);

    await waitFor(() => {
      expect(screen.getByText("0 days")).toBeInTheDocument();
    });
    expect(screen.getByText(/Latest pre-commit build failed/)).toBeInTheDocument();
  });
});
