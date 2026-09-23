import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { RecentRunsTable } from "./recent-runs-table";
import type { RunRecord } from "@/lib/types";

describe("RecentRunsTable", () => {
  it("renders an empty state when there are no runs", () => {
    render(<RecentRunsTable runs={[]} />);

    expect(screen.getByText("No runs recorded yet.")).toBeInTheDocument();
  });

  it("renders a passed run with its checks ratio", () => {
    const runs: RunRecord[] = [
      {
        timestamp: "2026-09-13T11:37:32-03:00",
        hook_type: "pre-commit",
        success: true,
        duration_ms: 1234,
        hooks: [
          { name: "Gitleaks", success: true, duration_ms: 500 },
          { name: "Pytest", success: true, duration_ms: 734 },
        ],
      },
    ];
    render(<RecentRunsTable runs={runs} />);

    expect(screen.getByText("pre-commit")).toBeInTheDocument();
    expect(screen.getByText(/passed/)).toBeInTheDocument();
    expect(screen.getByText("2/2")).toBeInTheDocument();
  });

  it("renders a failed run", () => {
    const runs: RunRecord[] = [
      {
        timestamp: "2026-09-13T11:37:32-03:00",
        hook_type: "pre-commit",
        success: false,
        duration_ms: 42,
        hooks: [{ name: "Fails", success: false, duration_ms: 42 }],
      },
    ];
    render(<RecentRunsTable runs={runs} />);

    expect(screen.getByText(/failed/)).toBeInTheDocument();
    expect(screen.getByText("0/1")).toBeInTheDocument();
  });

  it("tolerates a run with no hooks array", () => {
    const runs: RunRecord[] = [
      {
        timestamp: "2026-09-13T11:37:32-03:00",
        hook_type: "pre-commit",
        success: true,
        duration_ms: 10,
        hooks: undefined as unknown as RunRecord["hooks"],
      },
    ];
    render(<RecentRunsTable runs={runs} />);

    expect(screen.getByText("0/0")).toBeInTheDocument();
  });
});
