/**
 * Tests for VehicleCard component.
 *
 * Verifies:
 * - Renders vehicle name
 * - Shows raw device ID for admin users
 * - Shows masked device ID for non-admin users
 * - Clicking triggers onSelect callback
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { VehicleCard } from "@/components/VehicleCard";
import type { Vehicle } from "@/types/domain";

const baseVehicle: Vehicle = {
  id: "v-1",
  device_id: "ABCD1234EFGH",
  name: "Truck Alpha",
  
  created_at: "2026-01-01T00:00:00Z",
};

describe("VehicleCard", () => {
  it("renders vehicle name", () => {
    render(<VehicleCard vehicle={baseVehicle} role="admin" onSelect={vi.fn()} />);
    expect(screen.getByText("Truck Alpha")).toBeInTheDocument();
  });

  it("shows raw device ID for admin", () => {
    render(<VehicleCard vehicle={baseVehicle} role="admin" onSelect={vi.fn()} />);
    expect(screen.getByText("ABCD1234EFGH")).toBeInTheDocument();
  });

  it("shows masked device ID for non-admin user", () => {
    render(<VehicleCard vehicle={baseVehicle} role="user" onSelect={vi.fn()} />);
    // Masked format: DEV-****-{last4}
    expect(screen.getByText("DEV-****-EFGH")).toBeInTheDocument();
    expect(screen.queryByText("ABCD1234EFGH")).not.toBeInTheDocument();
  });

  it("calls onSelect with vehicle ID on click", () => {
    const onSelect = vi.fn();
    render(<VehicleCard vehicle={baseVehicle} role="admin" onSelect={onSelect} />);

    const card = screen.getByRole("button");
    fireEvent.click(card);
    expect(onSelect).toHaveBeenCalledWith("v-1");
  });

  it("masks short device IDs correctly", () => {
    const shortIdVehicle: Vehicle = { ...baseVehicle, device_id: "AB" };
    render(<VehicleCard vehicle={shortIdVehicle} role="user" onSelect={vi.fn()} />);
    expect(screen.getByText("DEV-****-????")).toBeInTheDocument();
  });
});
