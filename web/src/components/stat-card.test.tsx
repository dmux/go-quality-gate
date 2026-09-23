import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatCard } from "./stat-card";

describe("StatCard", () => {
  it("renders label, value, and icon", () => {
    render(<StatCard label="Streak" value="15 days" icon="🔥" />);

    expect(screen.getByText("Streak")).toBeInTheDocument();
    expect(screen.getByText("15 days")).toBeInTheDocument();
    expect(screen.getByText("🔥")).toBeInTheDocument();
  });

  it("renders a hint when provided", () => {
    render(<StatCard label="Streak" value="15 days" hint="without breaking pre-commit" />);

    expect(screen.getByText("without breaking pre-commit")).toBeInTheDocument();
  });

  it("omits the hint element when not provided", () => {
    const { container } = render(<StatCard label="Streak" value="15 days" />);

    expect(container.querySelector("p")).toBeNull();
  });

  it("omits the icon when not provided", () => {
    render(<StatCard label="Streak" value="15 days" />);

    expect(screen.queryByText("🔥")).not.toBeInTheDocument();
  });
});
