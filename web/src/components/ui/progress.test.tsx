import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  Progress,
  ProgressTrack,
  ProgressIndicator,
  ProgressLabel,
  ProgressValue,
} from "./progress";

describe("Progress family", () => {
  it("renders the composed Progress with a value", () => {
    render(<Progress value={42} data-testid="progress" />);

    expect(screen.getByTestId("progress")).toBeInTheDocument();
  });

  it("renders the individual sub-components directly", () => {
    render(
      <Progress value={50}>
        <ProgressLabel>Loading</ProgressLabel>
        <ProgressValue />
        <ProgressTrack>
          <ProgressIndicator />
        </ProgressTrack>
      </Progress>
    );

    expect(screen.getByText("Loading")).toBeInTheDocument();
  });
});
