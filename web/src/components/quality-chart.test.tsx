import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QualityChart, formatTooltipLabel, formatTooltipValue } from "./quality-chart";
import type { DailySummary } from "@/lib/types";

function makeDaily(days: number): DailySummary[] {
  return Array.from({ length: days }, (_, i) => ({
    date: `2026-09-${String(i + 1).padStart(2, "0")}`,
    total: i + 1,
    failed: i % 2,
    success_rate: 100 - i,
  }));
}

describe("QualityChart", () => {
  it("renders the title and both range tabs, defaulting to 30 days", () => {
    render(<QualityChart daily={makeDaily(30)} />);

    expect(screen.getByText("📈 Quality evolution")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "7 days" })).toBeInTheDocument();
    const thirtyDaysTab = screen.getByRole("tab", { name: "30 days" });
    expect(thirtyDaysTab).toHaveAttribute("aria-selected", "true");
  });

  it("switches to the 7-day range on click", async () => {
    const user = userEvent.setup();
    render(<QualityChart daily={makeDaily(30)} />);

    await user.click(screen.getByRole("tab", { name: "7 days" }));

    expect(screen.getByRole("tab", { name: "7 days" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tab", { name: "30 days" })).toHaveAttribute("aria-selected", "false");
  });

  it("renders with an empty daily series without crashing", () => {
    render(<QualityChart daily={[]} />);

    expect(screen.getByText("📈 Quality evolution")).toBeInTheDocument();
  });
});

describe("formatTooltipLabel", () => {
  it("returns the day's date from the payload", () => {
    expect(formatTooltipLabel(undefined, [{ payload: { date: "2026-09-13" } }])).toBe("2026-09-13");
  });

  it("returns an empty string when there's no payload", () => {
    expect(formatTooltipLabel(undefined, undefined)).toBe("");
    expect(formatTooltipLabel(undefined, [])).toBe("");
  });
});

describe("formatTooltipValue", () => {
  it("formats the success rate with a passed/total breakdown", () => {
    const item = { payload: { total: 4, failed: 1 } as DailySummary };
    expect(formatTooltipValue(75, "success_rate", item)).toBe("75% (3/4 runs)");
  });

  it("defaults to 0/0 when there's no payload", () => {
    expect(formatTooltipValue(0, "success_rate", undefined)).toBe("0% (0/0 runs)");
  });
});
