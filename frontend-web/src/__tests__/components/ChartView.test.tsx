/**
 * Tests for ChartView component.
 *
 * Verifies:
 * - Renders with sensor history data
 * - Shows fuel level and temperature labels
 * - Renders fallback when no history data
 * - Handles empty array gracefully
 */

import { describe, it, expect, vi } from "vitest";
import type { ReactNode } from "react";
import { render, screen } from "@testing-library/react";
import { ChartView } from "@/components/ChartView";
import type { SensorHistoryPoint } from "@/types/domain";

// Mock recharts to avoid SVG/rendering complexity in tests
vi.mock("recharts", () => ({
  LineChart: ({ children, data }: { children: ReactNode; data?: unknown[] }) => (
    <div data-testid="mock-line-chart" data-data-length={data?.length ?? 0}>
      {children}
    </div>
  ),
  Line: ({ name }: { name?: string }) => (
    <div data-testid="mock-line" data-name={name} />
  ),
  XAxis: () => <div data-testid="mock-x-axis" />,
  YAxis: ({ yAxisId }: { yAxisId?: string }) => (
    <div data-testid="mock-y-axis" data-axis={yAxisId} />
  ),
  CartesianGrid: () => <div data-testid="mock-cartesian-grid" />,
  Tooltip: () => <div data-testid="mock-tooltip" />,
  Legend: () => <div data-testid="mock-legend" />,
  ResponsiveContainer: ({ children }: { children: ReactNode }) => (
    <div data-testid="mock-responsive-container">{children}</div>
  ),
}));

const mockHistory: SensorHistoryPoint[] = [
  { timestamp: "2026-06-19T10:00:00Z", fuel_level: 80, temperature: 22 },
  { timestamp: "2026-06-19T10:05:00Z", fuel_level: 78, temperature: 23 },
  { timestamp: "2026-06-19T10:10:00Z", fuel_level: 76, temperature: 22.5 },
];

describe("ChartView", () => {
  it("renders chart with history data", () => {
    render(<ChartView history={mockHistory} />);
    expect(screen.getByTestId("mock-line-chart")).toBeInTheDocument();
  });

  it("renders both fuel and temperature lines", () => {
    render(<ChartView history={mockHistory} />);
    const lines = screen.getAllByTestId("mock-line");
    expect(lines).toHaveLength(2);
  });

  it("passes data points to the chart", () => {
    render(<ChartView history={mockHistory} />);
    const chart = screen.getByTestId("mock-line-chart");
    expect(chart.getAttribute("data-data-length")).toBe("3");
  });

  it("renders fuel level line with label", () => {
    render(<ChartView history={mockHistory} />);
    const lines = screen.getAllByTestId("mock-line");
    const fuelLine = lines.find((l) => l.getAttribute("data-name") === "Fuel %");
    expect(fuelLine).toBeDefined();
  });

  it("renders temperature line with label", () => {
    render(<ChartView history={mockHistory} />);
    const lines = screen.getAllByTestId("mock-line");
    const tempLine = lines.find((l) => l.getAttribute("data-name") === "Temp °C");
    expect(tempLine).toBeDefined();
  });

  it("shows fallback when history is empty", () => {
    render(<ChartView history={[]} />);
    expect(screen.getByText(/no history data available/i)).toBeInTheDocument();
  });

  it("shows fallback when history is undefined", () => {
    render(<ChartView history={undefined} />);
    expect(screen.getByText(/no history data available/i)).toBeInTheDocument();
  });
});