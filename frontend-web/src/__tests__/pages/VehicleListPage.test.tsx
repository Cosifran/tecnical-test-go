/**
 * Tests for VehicleListPage component.
 *
 * Verifies:
 * - Renders loading state while fetching
 * - Renders vehicle list after successful fetch
 * - Shows empty state when no vehicles
 * - Masks device IDs for non-admin users
 * - Shows raw device IDs for admin users
 * - Shows error message on fetch failure
 * - Calls fetchVehicles on mount
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { VehicleListPage } from "@/pages/VehicleListPage";
import { useFleetStore } from "@/stores/fleetStore";
import { useAuthStore } from "@/stores/authStore";
import type { Vehicle } from "@/types/domain";

const mockVehicle1: Vehicle = {
  id: "v-1",
  device_id: "ABCD1234EFGH",
  name: "Truck Alpha",
  
  created_at: "2026-01-01T00:00:00Z",
};

const mockVehicle2: Vehicle = {
  id: "v-2",
  device_id: "WXYZ5678IJKL",
  name: "Truck Beta",
  
  created_at: "2026-01-15T00:00:00Z",
};

// Mock fetchVehicles as no-op by default so tests can control state
const mockFetchVehicles = vi.fn().mockResolvedValue(undefined);

function renderVehicleListPage() {
  return render(
    <MemoryRouter initialEntries={["/vehicles"]}>
      <Routes>
        <Route path="/vehicles" element={<VehicleListPage />} />
        <Route path="/vehicles/:id" element={<div data-testid="detail-page">Detail</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe("VehicleListPage", () => {
  beforeEach(() => {
    // Reset store state, then set fetchVehicles to no-op for controlled tests
    useFleetStore.setState({
      vehicles: [],
      history: {},
      selectedVehicleId: null,
      isLoading: false,
      error: null,
      lastFetched: null,
      fetchVehicles: mockFetchVehicles,
    });
    localStorage.clear();
    vi.clearAllMocks();
    mockFetchVehicles.mockResolvedValue(undefined);
  });

  it("shows loading state while fetching vehicles", () => {
    useFleetStore.setState({ isLoading: true });
    useAuthStore.setState({ role: "admin", isAuthenticated: true });

    renderVehicleListPage();
    expect(screen.getByText(/loading vehicles/i)).toBeInTheDocument();
  });

  it("renders vehicles after successful fetch", () => {
    useFleetStore.setState({
      vehicles: [mockVehicle1, mockVehicle2],
      isLoading: false,
    });
    useAuthStore.setState({ role: "admin", isAuthenticated: true });

    renderVehicleListPage();
    expect(screen.getByText("Truck Alpha")).toBeInTheDocument();
    expect(screen.getByText("Truck Beta")).toBeInTheDocument();
  });

  it("shows raw device IDs for admin users", () => {
    useFleetStore.setState({
      vehicles: [mockVehicle1],
      isLoading: false,
    });
    useAuthStore.setState({ role: "admin", isAuthenticated: true });

    renderVehicleListPage();
    expect(screen.getByText("ABCD1234EFGH")).toBeInTheDocument();
  });

  it("shows masked device IDs for non-admin users", () => {
    useFleetStore.setState({
      vehicles: [mockVehicle1],
      isLoading: false,
    });
    useAuthStore.setState({ role: "user", isAuthenticated: true });

    renderVehicleListPage();
    expect(screen.getByText("DEV-****-EFGH")).toBeInTheDocument();
  });

  it("shows empty state when no vehicles", () => {
    useFleetStore.setState({ vehicles: [], isLoading: false });
    useAuthStore.setState({ role: "admin", isAuthenticated: true });

    renderVehicleListPage();
    expect(screen.getByText(/no vehicles are currently registered/i)).toBeInTheDocument();
  });

  it("shows error message on fetch failure", () => {
    useFleetStore.setState({
      vehicles: [],
      isLoading: false,
      error: "Network error",
    });
    useAuthStore.setState({ role: "admin", isAuthenticated: true });

    renderVehicleListPage();
    expect(screen.getByText(/network error/i)).toBeInTheDocument();
  });

  it("calls fetchVehicles on mount", async () => {
    useAuthStore.setState({ role: "admin", isAuthenticated: true });

    renderVehicleListPage();

    await waitFor(() => {
      expect(mockFetchVehicles).toHaveBeenCalled();
    });
  });
});
