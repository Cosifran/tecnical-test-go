/**
 * Tests for OfflineBanner component.
 *
 * Verifies:
 * - Not visible when online
 * - Shows banner with correct message when offline
 * - Disappears when connection restored
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, act } from "@testing-library/react";
import { OfflineBanner } from "@/components/OfflineBanner";

// Mock useOnline hook
const mockUseOnline = vi.fn();
vi.mock("@/hooks/useOnline", () => ({
  useOnline: () => mockUseOnline(),
}));

describe("OfflineBanner", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("does not render when online", () => {
    mockUseOnline.mockReturnValue(true);
    const { container } = render(<OfflineBanner />);
    expect(container.querySelector("[role='alert']")).toBeNull();
  });

  it("shows offline banner when offline", () => {
    mockUseOnline.mockReturnValue(false);
    render(<OfflineBanner />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByText(/you are offline/i)).toBeInTheDocument();
    expect(screen.getByText(/showing cached data/i)).toBeInTheDocument();
  });

  it("disappears when connection is restored", () => {
    // Start offline
    mockUseOnline.mockReturnValue(false);
    const { rerender } = render(<OfflineBanner />);
    expect(screen.getByRole("alert")).toBeInTheDocument();

    // Go online
    mockUseOnline.mockReturnValue(true);
    rerender(<OfflineBanner />);
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("has sticky positioning class", () => {
    mockUseOnline.mockReturnValue(false);
    render(<OfflineBanner />);
    const banner = screen.getByRole("alert");
    expect(banner.className).toContain("sticky");
  });
});