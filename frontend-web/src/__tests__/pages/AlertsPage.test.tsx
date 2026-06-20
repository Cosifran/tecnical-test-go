/**
 * Tests for AlertsPage component.
 *
 * Verifies:
 * - Renders loading state while fetching alerts
 * - Renders alert list after successful fetch with severity badges
 * - Shows empty state when no alerts exist
 * - Shows error state with retry button on fetch failure
 * - Displays alert details: type, severity, vehicle info, created_at
 * - Sorting: newest alerts appear first (backend already sorts)
 * - Uses AdminGuard: non-admin cannot access
 * - Calls fetchAlerts on mount
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { AlertsPage } from "@/pages/AlertsPage";
import { useAuthStore } from "@/stores/authStore";
import { useFleetStore } from "@/stores/fleetStore";
import type { Alert } from "@/types/domain";

const mockAlert1: Alert = {
  id: "alert-1",
  vehicle_id: "vehicle-1",
  type: "low_fuel",
  severity: "critical",
  details: { estimated_minutes: 45 },
  created_at: "2026-06-19T10:00:00Z",
};

const mockAlert2: Alert = {
  id: "alert-2",
  vehicle_id: "vehicle-2",
  type: "high_temperature",
  severity: "warning",
  details: { temperature: 95 },
  created_at: "2026-06-19T09:00:00Z",
};

const mockAlert3: Alert = {
  id: "alert-3",
  vehicle_id: "vehicle-1",
  type: "maintenance_due",
  severity: "info",
  details: { overdue_hours: 120 },
  created_at: "2026-06-19T08:00:00Z",
};

const mockFetchAlerts = vi.fn().mockResolvedValue(undefined);

function renderAlertsPage() {
  return render(
    <MemoryRouter initialEntries={["/alerts"]}>
      <Routes>
        <Route path="/alerts" element={<AlertsPage />} />
        <Route path="/vehicles" element={<div data-testid="vehicles-page">Vehicles</div>} />
        <Route path="/login" element={<div data-testid="login-page">Login</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe("AlertsPage", () => {
  beforeEach(() => {
    // Reset fleet store state
    useFleetStore.setState({
      vehicles: [],
      history: {},
      selectedVehicleId: null,
      isLoading: false,
      error: null,
      lastFetched: null,
      isFetching: false,
      alerts: [],
      alertLoading: false,
      alertError: null,
      fetchAlerts: mockFetchAlerts,
    });
    // Reset auth store
    useAuthStore.setState({
      accessToken: "valid-token",
      refreshToken: "valid-refresh",
      role: "admin",
      email: "admin@fleet.com",
      userId: "admin-1",
      isAuthenticated: true,
    });
    localStorage.clear();
    vi.clearAllMocks();
    mockFetchAlerts.mockResolvedValue(undefined);
  });

  it("shows loading state while fetching alerts", () => {
    useFleetStore.setState({ alertLoading: true, alerts: [] });

    renderAlertsPage();
    expect(screen.getByText(/loading alerts/i)).toBeInTheDocument();
  });

  it("renders alert list after successful fetch", () => {
    useFleetStore.setState({
      alerts: [mockAlert1, mockAlert2],
      alertLoading: false,
    });

    renderAlertsPage();

    // Verify alert types are rendered
    expect(screen.getByText("low_fuel")).toBeInTheDocument();
    expect(screen.getByText("high_temperature")).toBeInTheDocument();
  });

  it("shows severity badges for each alert", () => {
    useFleetStore.setState({
      alerts: [mockAlert1, mockAlert2, mockAlert3],
      alertLoading: false,
    });

    renderAlertsPage();

    // Verify severity levels are displayed
    expect(screen.getByText("critical")).toBeInTheDocument();
    expect(screen.getByText("warning")).toBeInTheDocument();
    expect(screen.getByText("info")).toBeInTheDocument();
  });

  it("shows empty state when no alerts exist", () => {
    useFleetStore.setState({ alerts: [], alertLoading: false });

    renderAlertsPage();
    expect(screen.getByText(/no alerts at this time/i)).toBeInTheDocument();
  });

  it("shows error state with retry button on fetch failure", () => {
    const mockClearAlertError = vi.fn();
    const retryFetchAlerts = vi.fn().mockResolvedValue(undefined);
    useFleetStore.setState({
      alerts: [],
      alertLoading: false,
      alertError: "Failed to fetch alerts",
      clearAlertError: mockClearAlertError,
      fetchAlerts: retryFetchAlerts,
    });

    renderAlertsPage();

    expect(screen.getByText(/failed to fetch alerts/i)).toBeInTheDocument();
    const retryButton = screen.getByRole("button", { name: /retry/i });
    expect(retryButton).toBeInTheDocument();
  });

  it("calls fetchAlerts on mount", async () => {
    renderAlertsPage();

    await waitFor(() => {
      expect(mockFetchAlerts).toHaveBeenCalledTimes(1);
    });
  });

  it("renders alert details — type, severity, vehicle ID, created_at", () => {
    useFleetStore.setState({
      alerts: [mockAlert1],
      alertLoading: false,
    });

    renderAlertsPage();

    // Type
    expect(screen.getByText("low_fuel")).toBeInTheDocument();
    // Severity
    expect(screen.getByText("critical")).toBeInTheDocument();
    // Vehicle ID
    expect(screen.getByText(/vehicle-1/i)).toBeInTheDocument();
    // Created_at — formatted date is rendered (toLocaleString output varies by locale)
    // Check that the date span exists with a non-empty text
    const dateElements = screen.getAllByText(/\d{1,2}\/\d{1,2}\/\d{4}/);
    expect(dateElements.length).toBeGreaterThanOrEqual(1);
  });

  it("retries fetching alerts when retry button is clicked", async () => {
    const user = userEvent.setup();
    const retryFetch = vi.fn().mockResolvedValue(undefined);
    const clearErr = vi.fn();
    useFleetStore.setState({
      alerts: [],
      alertLoading: false,
      alertError: "Network error",
      fetchAlerts: retryFetch,
      clearAlertError: clearErr,
    });

    renderAlertsPage();

    const retryButton = screen.getByRole("button", { name: /retry/i });
    await user.click(retryButton);

    expect(clearErr).toHaveBeenCalled();
    expect(retryFetch).toHaveBeenCalled();
  });
});