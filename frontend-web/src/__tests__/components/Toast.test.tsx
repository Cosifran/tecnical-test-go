/**
 * Tests for Toast component.
 *
 * Verifies:
 * - Shows toast with message and estimated minutes
 * - Has dismiss button
 * - Calls onDismiss when dismiss clicked
 * - ToastContainer renders all toasts from store
 * - ToastContainer removes toast on dismiss
 * - Auto-dismiss after 5 seconds
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, act, fireEvent } from "@testing-library/react";
import { Toast, ToastContainer } from "@/components/Toast";
import { useToastStore } from "@/stores/toastStore";

describe("Toast", () => {
  beforeEach(() => {
    useToastStore.getState().clearToasts();
  });

  describe("single Toast", () => {
    it("renders low_fuel toast with message and estimated minutes", () => {
      render(
        <Toast
          id="toast-1"
          type="low_fuel"
          message="Low fuel alert for vehicle v-001"
          vehicleId="v-001"
          estimatedMinutes={45}
          createdAt={Date.now()}
          onDismiss={() => {}}
        />
      );

      expect(screen.getByText(/Low fuel alert for vehicle v-001/i)).toBeInTheDocument();
      expect(screen.getByText(/45 minutes/i)).toBeInTheDocument();
    });

    it("has a dismiss button", () => {
      render(
        <Toast
          id="toast-1"
          type="low_fuel"
          message="Low fuel alert"
          createdAt={Date.now()}
          onDismiss={() => {}}
        />
      );

      expect(screen.getByRole("button", { name: /dismiss/i })).toBeInTheDocument();
    });

    it("calls onDismiss when dismiss button is clicked", () => {
      const onDismiss = vi.fn();
      render(
        <Toast
          id="toast-1"
          type="low_fuel"
          message="Low fuel alert"
          createdAt={Date.now()}
          onDismiss={onDismiss}
        />
      );

      fireEvent.click(screen.getByRole("button", { name: /dismiss/i }));
      expect(onDismiss).toHaveBeenCalledWith("toast-1");
    });
  });

  describe("ToastContainer", () => {
    it("does not render when no toasts", () => {
      const { container } = render(<ToastContainer />);
      expect(container.querySelector("[data-testid='toast-container']")).toBeNull();
    });

    it("renders all toasts from toastStore", () => {
      useToastStore.getState().addToast({
        type: "low_fuel",
        message: "Low fuel for v-001",
        vehicleId: "v-001",
        estimatedMinutes: 45,
      });
      useToastStore.getState().addToast({
        type: "low_fuel",
        message: "Low fuel for v-002",
        vehicleId: "v-002",
        estimatedMinutes: 30,
      });

      render(<ToastContainer />);

      expect(screen.getByText(/Low fuel for v-001/i)).toBeInTheDocument();
      expect(screen.getByText(/Low fuel for v-002/i)).toBeInTheDocument();
    });

    it("removes toast when dismiss is clicked", () => {
      useToastStore.getState().addToast({
        type: "low_fuel",
        message: "Low fuel for v-003",
        vehicleId: "v-003",
        estimatedMinutes: 15,
      });

      render(<ToastContainer />);
      expect(screen.getByText(/Low fuel for v-003/i)).toBeInTheDocument();

      fireEvent.click(screen.getByRole("button", { name: /dismiss/i }));

      expect(screen.queryByText(/Low fuel for v-003/i)).not.toBeInTheDocument();
    });

    it("auto-dismisses toasts after 5 seconds", () => {
      vi.useFakeTimers();

      useToastStore.getState().addToast({
        type: "low_fuel",
        message: "Low fuel for v-004",
        vehicleId: "v-004",
        estimatedMinutes: 10,
      });

      render(<ToastContainer />);
      expect(screen.getByText(/Low fuel for v-004/i)).toBeInTheDocument();

      act(() => {
        vi.advanceTimersByTime(5000);
      });

      expect(screen.queryByText(/Low fuel for v-004/i)).not.toBeInTheDocument();

      vi.useRealTimers();
    });
  });
});