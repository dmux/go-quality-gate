import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

// next/font/google requires Next's build-time SWC transform to work; under
// plain Vitest it isn't a real font loader, so stub it with the same shape
// (an object carrying a CSS variable class name) our layout consumes.
vi.mock("next/font/google", () => ({
  Geist: () => ({ variable: "mock-geist-sans" }),
  Geist_Mono: () => ({ variable: "mock-geist-mono" }),
}));

const { default: RootLayout } = await import("./layout");

describe("RootLayout", () => {
  it("renders its children and applies the dark class to <html>", () => {
    render(
      <RootLayout params={Promise.resolve({})}>
        <p>hello from child</p>
      </RootLayout>
    );

    expect(screen.getByText("hello from child")).toBeInTheDocument();
    expect(document.documentElement.className).toContain("dark");
  });
});
