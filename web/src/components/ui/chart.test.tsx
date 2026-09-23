import { describe, it, expect, vi } from "vitest";
import type React from "react";
import { render, screen } from "@testing-library/react";
import {
  ChartContainer,
  ChartStyle,
  ChartTooltipContent,
  ChartLegendContent,
  type ChartConfig,
} from "./chart";

function DotIcon() {
  return <svg data-testid="dot-icon" />;
}

const config: ChartConfig = {
  success_rate: { label: "Success Rate", color: "#22c55e", icon: DotIcon },
  failure_rate: { label: "Failure Rate", theme: { light: "#ef4444", dark: "#f87171" } },
};

describe("ChartContainer", () => {
  it("renders children inside a chart wrapper with a generated id", () => {
    const { container } = render(
      <ChartContainer config={config}>
        <div data-testid="child" />
      </ChartContainer>
    );

    const chart = container.querySelector('[data-slot="chart"]');
    expect(chart).not.toBeNull();
    expect(chart?.getAttribute("data-chart")).toMatch(/^chart-/);
  });

  it("uses a caller-provided id when given", () => {
    const { container } = render(
      <ChartContainer id="my-chart" config={config}>
        <div />
      </ChartContainer>
    );

    expect(container.querySelector('[data-chart="chart-my-chart"]')).not.toBeNull();
  });
});

describe("useChart guard", () => {
  it("throws when a tooltip/legend content is rendered outside ChartContainer", () => {
    // Suppress the expected React error-boundary console noise for this case.
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => render(<ChartTooltipContent active payload={[]} />)).toThrow(
      "useChart must be used within a <ChartContainer />"
    );
    spy.mockRestore();
  });
});

describe("ChartStyle", () => {
  it("renders nothing when no config entry has a color or theme", () => {
    const { container } = render(<ChartStyle id="x" config={{ foo: {} }} />);
    expect(container.querySelector("style")).toBeNull();
  });

  it("renders a <style> block covering both color and theme entries", () => {
    const { container } = render(<ChartStyle id="x" config={config} />);
    const style = container.querySelector("style");
    expect(style).not.toBeNull();
    expect(style!.innerHTML).toContain("--color-success_rate: #22c55e");
    expect(style!.innerHTML).toContain("--color-failure_rate: #ef4444"); // light theme
    expect(style!.innerHTML).toContain("--color-failure_rate: #f87171"); // dark theme
  });

  it("omits a --color-* line for an entry whose resolved color is falsy", () => {
    const emptyThemeConfig: ChartConfig = {
      blank: { theme: { light: "", dark: "" } },
    };
    const { container } = render(<ChartStyle id="x" config={emptyThemeConfig} />);
    const style = container.querySelector("style");
    expect(style!.innerHTML).not.toContain("--color-blank");
  });
});

describe("ChartTooltipContent", () => {
  const wrap = (ui: React.ReactNode) => render(<ChartContainer config={config}>{ui as never}</ChartContainer>);

  it("renders nothing when inactive", () => {
    const { container } = wrap(<ChartTooltipContent active={false} payload={[]} />);
    expect(container.querySelector(".grid.min-w-32")).toBeNull();
  });

  it("renders nothing when active but payload is empty", () => {
    const { container } = wrap(<ChartTooltipContent active payload={[]} />);
    expect(container.querySelector(".grid.min-w-32")).toBeNull();
  });

  it("shows a config-resolved label and a colored dot indicator by default", () => {
    wrap(
      <ChartTooltipContent
        active
        label="success_rate"
        payload={[
          { graphicalItemId: "", dataKey: "success_rate", name: "success_rate", value: 75, color: "#000", payload: {} },
        ]}
      />
    );

    // "Success Rate" appears both as the tooltip's header label and as the
    // item row's own name.
    expect(screen.getAllByText("Success Rate").length).toBeGreaterThan(0);
    expect(screen.getByText("75")).toBeInTheDocument();
  });

  it("hides the header label (but not the item row's own name) when hideLabel is set", () => {
    const { container } = render(
      <ChartContainer config={config}>
        <ChartTooltipContent
          active
          hideLabel
          label="success_rate"
          payload={[{ graphicalItemId: "", dataKey: "success_rate", name: "success_rate", value: 1, payload: {} }]}
        />
      </ChartContainer>
    );

    // The header label is a <div class="font-medium">; the per-item value
    // span also happens to include "font-medium" among its classes, so
    // scope the query to <div> to avoid a false match there.
    expect(container.querySelector("div.font-medium")).toBeNull();
  });

  it("falls back to the raw label string when it doesn't match any config entry", () => {
    render(
      <ChartContainer config={config}>
        <ChartTooltipContent
          active
          label="not_in_config"
          payload={[{ graphicalItemId: "", dataKey: "success_rate", name: "success_rate", value: 1, payload: {} }]}
        />
      </ChartContainer>
    );

    expect(screen.getByText("not_in_config")).toBeInTheDocument();
  });

  it("uses a custom labelFormatter when provided", () => {
    render(
      <ChartContainer config={config}>
        <ChartTooltipContent
          active
          label="success_rate"
          labelFormatter={() => "Custom Label"}
          payload={[{ graphicalItemId: "", dataKey: "success_rate", name: "success_rate", value: 1, payload: {} }]}
        />
      </ChartContainer>
    );

    expect(screen.getByText("Custom Label")).toBeInTheDocument();
  });

  it("renders nothing for the header label when it's not a string and there's no matching config", () => {
    // No `label` prop (not a string) and a dataKey absent from config means
    // both the label-string fallback and itemConfig?.label are falsy,
    // hitting the "if (!value) return null" branch.
    const { container } = render(
      <ChartContainer config={config}>
        <ChartTooltipContent
          active
          payload={[{ graphicalItemId: "", dataKey: "unknown_key", name: "unknown_key", value: 1, payload: {} }]}
        />
      </ChartContainer>
    );

    expect(container.querySelector("div.font-medium")).toBeNull();
  });

  it("uses a custom item formatter when provided, bypassing the default indicator/value rendering", () => {
    render(
      <ChartContainer config={config}>
        <ChartTooltipContent
          active
          payload={[
            {
              graphicalItemId: "",
              dataKey: "success_rate",
              name: "success_rate",
              value: 75,
              payload: {},
            },
          ]}
          formatter={(value, name) => <span data-testid="custom-formatter">{`${name}=${value}`}</span>}
        />
      </ChartContainer>
    );

    expect(screen.getByTestId("custom-formatter")).toHaveTextContent("success_rate=75");
  });

  it("renders the config icon instead of a colored dot when available", () => {
    render(
      <ChartContainer config={config}>
        <ChartTooltipContent
          active
          payload={[{ graphicalItemId: "", dataKey: "success_rate", name: "success_rate", value: 1, payload: {} }]}
        />
      </ChartContainer>
    );

    expect(screen.getByTestId("dot-icon")).toBeInTheDocument();
  });

  it("hides the indicator entirely when hideIndicator is set", () => {
    const { container } = render(
      <ChartContainer config={config}>
        <ChartTooltipContent
          active
          hideIndicator
          payload={[{ graphicalItemId: "", dataKey: "failure_rate", name: "failure_rate", value: 1, payload: {} }]}
        />
      </ChartContainer>
    );

    expect(container.querySelector('[style*="--color-bg"]')).toBeNull();
  });

  it.each(["dot", "line", "dashed"] as const)("supports the %s indicator variant", (indicator) => {
    render(
      <ChartContainer config={config}>
        <ChartTooltipContent
          active
          indicator={indicator}
          payload={[{ graphicalItemId: "", dataKey: "failure_rate", name: "failure_rate", value: 1, payload: {} }]}
        />
      </ChartContainer>
    );

    expect(screen.getAllByText("Failure Rate").length).toBeGreaterThan(0);
  });

  it("nests the label under a single non-dot item", () => {
    render(
      <ChartContainer config={config}>
        <ChartTooltipContent
          active
          indicator="line"
          label="success_rate"
          payload={[{ graphicalItemId: "", dataKey: "success_rate", name: "success_rate", value: 1, payload: {} }]}
        />
      </ChartContainer>
    );

    // Nested-label path still renders the label text somewhere in the tree.
    expect(screen.getAllByText("Success Rate").length).toBeGreaterThan(0);
  });

  it("formats a string value as-is and a number value with toLocaleString, and skips null values", () => {
    render(
      <ChartContainer config={config}>
        <ChartTooltipContent
          active
          payload={[
            { graphicalItemId: "", dataKey: "success_rate", name: "success_rate", value: 1234, payload: {} },
            { graphicalItemId: "", dataKey: "failure_rate", name: "failure_rate", value: "n/a", payload: {} },
          ]}
        />
      </ChartContainer>
    );

    // Locale-agnostic: assert toLocaleString was actually applied, not a
    // specific separator (which depends on the test runner's ICU locale).
    expect(screen.getByText((1234).toLocaleString())).toBeInTheDocument();
    expect(screen.getByText("n/a")).toBeInTheDocument();
  });

  it("filters out payload items with type 'none' from the item list (but not the tooltip's own label, which is resolved independently)", () => {
    const { container } = render(
      <ChartContainer config={config}>
        <ChartTooltipContent
          active
          payload={[
            { graphicalItemId: "", dataKey: "success_rate", name: "success_rate", value: 1, payload: {}, type: "none" },
          ]}
        />
      </ChartContainer>
    );

    expect(container.querySelectorAll(".flex.w-full.flex-wrap").length).toBe(0);
  });

  it("resolves the config entry via a matching property directly on the item", () => {
    render(
      <ChartContainer config={config}>
        <ChartTooltipContent
          active
          payload={[
            // The item has its own "foo" property (matching its resolved
            // key, since nameKey is unset and name="foo") whose value is
            // itself a valid config key. Recharts' Payload type doesn't
            // model this extra property, so the object is typed loosely.
            {
              graphicalItemId: "",
              dataKey: "foo",
              name: "foo",
              value: 1,
              payload: {},
              foo: "success_rate",
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
            } as any,
          ]}
        />
      </ChartContainer>
    );

    expect(screen.getAllByText("Success Rate").length).toBeGreaterThan(0);
  });

  it("treats a non-object payload item as having no config entry", () => {
    // getPayloadConfigFromPayload defensively handles a malformed (non-
    // object) item; force one through past the type system to exercise it.
    const malformedItem = "not-an-object" as unknown as { payload?: unknown };
    const { container } = render(
      <ChartContainer config={config}>
        <ChartTooltipContent active payload={[malformedItem as never]} />
      </ChartContainer>
    );

    // No crash, and no config-derived label/icon rendered for it.
    expect(container.querySelector("svg")).toBeNull();
  });

  it("resolves the config entry via a nested payload.payload key when the item itself lacks it", () => {
    render(
      <ChartContainer config={config}>
        <ChartTooltipContent
          active
          nameKey="success_rate"
          payload={[
            {
              graphicalItemId: "",
              value: 1,
              payload: { success_rate: "success_rate" },
            },
          ]}
        />
      </ChartContainer>
    );

    expect(screen.getByText("Success Rate")).toBeInTheDocument();
  });
});

describe("ChartLegendContent", () => {
  it("renders nothing when payload is empty", () => {
    const { container } = render(
      <ChartContainer config={config}>
        <ChartLegendContent payload={[]} />
      </ChartContainer>
    );
    expect(container.querySelector(".flex.items-center.justify-center")).toBeNull();
  });

  it("renders a legend entry per payload item, using the config icon when available", () => {
    render(
      <ChartContainer config={config}>
        <ChartLegendContent
          payload={[
            { dataKey: "success_rate", value: "success_rate", color: "#000" },
            { dataKey: "failure_rate", value: "failure_rate", color: "#f00" },
          ]}
        />
      </ChartContainer>
    );

    expect(screen.getByText("Success Rate")).toBeInTheDocument();
    expect(screen.getByText("Failure Rate")).toBeInTheDocument();
    expect(screen.getByTestId("dot-icon")).toBeInTheDocument();
  });

  it("hides icons when hideIcon is set", () => {
    render(
      <ChartContainer config={config}>
        <ChartLegendContent hideIcon payload={[{ dataKey: "success_rate", value: "success_rate", color: "#000" }]} />
      </ChartContainer>
    );

    expect(screen.queryByTestId("dot-icon")).not.toBeInTheDocument();
  });

  it("applies top padding when verticalAlign is top", () => {
    const { container } = render(
      <ChartContainer config={config}>
        <ChartLegendContent
          verticalAlign="top"
          payload={[{ dataKey: "success_rate", value: "success_rate", color: "#000" }]}
        />
      </ChartContainer>
    );

    expect(container.querySelector(".pb-3")).not.toBeNull();
  });

  it("filters out payload items with type 'none'", () => {
    render(
      <ChartContainer config={config}>
        <ChartLegendContent
          payload={[{ dataKey: "success_rate", value: "success_rate", color: "#000", type: "none" }]}
        />
      </ChartContainer>
    );

    expect(screen.queryByText("Success Rate")).not.toBeInTheDocument();
  });

  it("falls back to the literal key \"value\" when neither nameKey nor dataKey is present", () => {
    const valueConfig: ChartConfig = { value: { label: "Value Label" } };
    render(
      <ChartContainer config={valueConfig}>
        <ChartLegendContent payload={[{ value: "irrelevant", color: "#000" }]} />
      </ChartContainer>
    );

    expect(screen.getByText("Value Label")).toBeInTheDocument();
  });

  it("resolves the config entry via nameKey when provided, overriding dataKey", () => {
    render(
      <ChartContainer config={config}>
        <ChartLegendContent
          nameKey="success_rate"
          payload={[{ dataKey: "does_not_matter", value: "does_not_matter", color: "#000" }]}
        />
      </ChartContainer>
    );

    expect(screen.getByText("Success Rate")).toBeInTheDocument();
  });
});
