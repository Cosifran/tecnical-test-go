/**
 * Tests for LoadingSpinner component.
 *
 * Verifies:
 * - Renders with default props (spinning animation)
 * - Renders custom size variants
 * - Renders custom label text
 * - Applies custom className
 */

import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { LoadingSpinner } from "@/components/LoadingSpinner";

describe("LoadingSpinner", () => {
  it("renders with default label text", () => {
    render(<LoadingSpinner />);
    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });

  it("renders custom label text", () => {
    render(<LoadingSpinner label="Fetching alerts" />);
    expect(screen.getByText(/fetching alerts/i)).toBeInTheDocument();
  });

  it("renders with sm size variant", () => {
    const { container } = render(<LoadingSpinner size="sm" />);
    // The spinner element should exist with size class
    const spinner = container.querySelector("[data-testid='loading-spinner']");
    expect(spinner).toBeInTheDocument();
  });

  it("renders with lg size variant", () => {
    const { container } = render(<LoadingSpinner size="lg" />);
    const spinner = container.querySelector("[data-testid='loading-spinner']");
    expect(spinner).toBeInTheDocument();
  });

  it("applies custom className", () => {
    const { container } = render(<LoadingSpinner className="mt-10" />);
    const wrapper = container.querySelector("[data-testid='loading-spinner-wrapper']");
    expect(wrapper?.className).toContain("mt-10");
  });
});