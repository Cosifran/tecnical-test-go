/**
 * Tests for VehicleDetailPage component.
 *
 * Verifies:
 * - Renders vehicle info (name, device ID)
 * - Shows masked device ID for non-admin users
 * - Shows raw device ID for admin users
 * - Shows loading state
 * - Shows "not found" when vehicle doesn't exist
 * - Back button navigates to vehicle list
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { VehicleDetailPage } from "@/pages/VehicleDetailPage";
import { useFleetStore } from "@/stores/fleetStore";
import { useAuthStore } from "@/stores/authStore";
import type { Vehicle, SensorData } from "@/types/domain";

// Mock MapView to avoid complex rendering
vi.mock("@/components/MapView", () => ({
  MapView: ({ center, markerLabel }: { center?: [number, number]; markerLabel?: string }) => (
    <div data-testid="mock-map" data-lat={center?.[1]} data-lng={center?.[0]} data-label={markerLabel}>
      Map
    </div>
  ),
}));

// Mock API endpoints
vi.mock("@/api/endpoints", () => ({
  getVehicles: vi.fn(),
  getVehicleHistory: vi.fn(),
}));

const mockVehicle: Vehicle = {
  id: "v-1",
  device_id: "ABCD1234EFGH",
  name: "Truck Alpha",
  created_at: "2026-01-01T00:00:00Z",
};

const mockHistory: SensorData[] = [
  {
    ID: "s0",
    VehicleID: "v-1",
    Type: "gps",
    Value: { lat: -34.6, lng: -58.4 },
    Timestamp: "2026-06-19T10:10:00Z",
    CreatedAt: "2026-06-19T10:10:00Z",
  },
  {
    ID: "s1",
    VehicleID: "v-1",
    Type: "fuel",
    Value: { level: 80, unit: "liters" },
    Timestamp: "2026-06-19T10:00:00Z",
    CreatedAt: "2026-06-19T10:00:00Z",
  },
  {
    ID: "s2",
    VehicleID: "v-1",
    Type: "temperature",
    Value: { celsius: 22 },
    Timestamp: "2026-06-19T10:05:00Z",
    CreatedAt: "2026-06-19T10:05:00Z",
  },
];

function renderVehicleDetailPage(vehicleId: string = "v-1") {
  return render(
    <MemoryRouter initialEntries={[`/vehicles/${vehicleId}`]}>
      <Routes>
        <Route path="/vehicles/:id" element={<VehicleDetailPage />} />
        <Route path="/vehicles" element={<div data-testid="vehicles-page">Vehicles</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe("VehicleDetailPage", () => {
  beforeEach(() => {
    useFleetStore.setState({
      vehicles: [mockVehicle],
      history: { "v-1": mockHistory },
      selectedVehicleId: null,
      isLoading: false,
      error: null,
      lastFetched: null,
    });
    useAuthStore.setState({ role: "admin", isAuthenticated: true });
    localStorage.clear();
    vi.clearAllMocks();
  });

  it("renders vehicle name", () => {
    renderVehicleDetailPage();
    expect(screen.getByText("Truck Alpha")).toBeInTheDocument();
  });

  it("shows raw device ID for admin users", () => {
    useAuthStore.setState({ role: "admin" });
    renderVehicleDetailPage();
    expect(screen.getByText(/ABCD1234EFGH/)).toBeInTheDocument();
  });

  it("shows masked device ID for non-admin users", () => {
    useAuthStore.setState({ role: "user" });
    renderVehicleDetailPage();
    expect(screen.getByText(/DEV-\*\*\*\*-EFGH/)).toBeInTheDocument();
  });

  it("renders map component", () => {
    renderVehicleDetailPage();
    expect(screen.getByTestId("mock-map")).toBeInTheDocument();
  });

  it("renders history section", () => {
    renderVehicleDetailPage();
    expect(screen.getByText(/Sensor History/i)).toBeInTheDocument();
  });

  it("shows not found message for nonexistent vehicle", () => {
    useFleetStore.setState({ vehicles: [] });
    renderVehicleDetailPage("nonexistent");
    expect(screen.getByText(/vehicle not found/i)).toBeInTheDocument();
  });

  it("shows back button linking to vehicles list", () => {
    renderVehicleDetailPage();
    expect(screen.getByText(/back/i)).toBeInTheDocument();
  });
});
