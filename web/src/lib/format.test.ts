import { describe, it, expect } from "vitest";
import { formatDuration, formatMinutesSaved, formatDayLabel, formatTimestamp } from "./format";

describe("formatDuration", () => {
  it("formats sub-second durations in ms", () => {
    expect(formatDuration(5)).toBe("5ms");
    expect(formatDuration(999)).toBe("999ms");
  });

  it("rounds non-integer sub-second durations (e.g. averaged values)", () => {
    expect(formatDuration(114.333333333333)).toBe("114ms");
    expect(formatDuration(8.66666666666666)).toBe("9ms");
  });

  it("formats sub-minute durations in seconds", () => {
    expect(formatDuration(1500)).toBe("1.5s");
    expect(formatDuration(59999)).toBe("60.0s");
  });

  it("formats minute-plus durations as Xm Ys", () => {
    expect(formatDuration(60000)).toBe("1m 0s");
    expect(formatDuration(125000)).toBe("2m 5s");
  });
});

describe("formatMinutesSaved", () => {
  it("formats sub-hour durations in minutes", () => {
    expect(formatMinutesSaved(5)).toBe("5 min");
    expect(formatMinutesSaved(59)).toBe("59 min");
  });

  it("formats hour-plus durations in hours", () => {
    expect(formatMinutesSaved(60)).toBe("1.0 h");
    expect(formatMinutesSaved(150)).toBe("2.5 h");
  });
});

describe("formatDayLabel", () => {
  it("formats a YYYY-MM-DD string as MM/DD", () => {
    expect(formatDayLabel("2026-09-13")).toBe("09/13");
  });
});

describe("formatTimestamp", () => {
  it("formats an ISO timestamp with date and time", () => {
    const out = formatTimestamp("2026-09-13T11:37:32-03:00");
    expect(out).toMatch(/09\/13/);
  });
});
