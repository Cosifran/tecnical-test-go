/**
 * Tests for MapView component.
 *
 * Since MapLibre GL / react-map-gl depends on browser APIs (WebGL, canvas),
 * we mock the Map component and test:
 * - Renders map with default center
 * - Renders with custom center coordinates
 */

import { describe, it, expect, vi } from "vitest";
import type { ReactNode } from "react";
import { render, screen } from "@testing-library/react";
import { MapView } from "@/components/MapView";

// Mock react-map-gl/maplibre to avoid WebGL/MapLibre dependency issues
vi.mock("react-map-gl/maplibre", () => ({
  Map: ({ children, initialViewState }: { children: ReactNode; initialViewState?: { longitude?: number; latitude?: number; zoom?: number } }) => (
    <div
      data-testid="mock-map"
      data-longitude={initialViewState?.longitude}
      data-latitude={initialViewState?.latitude}
      data-zoom={initialViewState?.zoom}
    >
      {children}
    </div>
  ),
  Marker: ({ children }: { children: ReactNode }) => <div data-testid="mock-marker">{children}</div>,
  NavigationControl: () => <div data-testid="mock-nav-control" />,
}));

describe("MapView", () => {
  it("renders a map", () => {
    render(<MapView />);
    expect(screen.getByTestId("mock-map")).toBeInTheDocument();
  });

  it("centers on default coordinates (Buenos Aires)", () => {
    render(<MapView />);
    const map = screen.getByTestId("mock-map");
    expect(map.getAttribute("data-latitude")).toBe("-34.6037");
    expect(map.getAttribute("data-longitude")).toBe("-58.3816");
  });

  it("centers on custom coordinates when provided", () => {
    render(<MapView center={[-74.0817, 4.6097]} zoom={10} />);
    const map = screen.getByTestId("mock-map");
    expect(map.getAttribute("data-latitude")).toBe("4.6097");
    expect(map.getAttribute("data-longitude")).toBe("-74.0817");
    expect(map.getAttribute("data-zoom")).toBe("10");
  });
});
