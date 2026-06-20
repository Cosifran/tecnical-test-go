/**
 * Tests for AuthGuard and AdminGuard route guard components.
 *
 * AuthGuard: redirects unauthenticated users to /login
 * AdminGuard: redirects non-admin users to /vehicles
 *
 * These tests verify:
 * - Authenticated users see children (AuthGuard)
 * - Unauthenticated users are redirected to /login
 * - Admin users see children (AdminGuard)
 * - Non-admin users are redirected to /vehicles
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { AuthGuard } from "@/guards/AuthGuard";
import { AdminGuard } from "@/guards/AdminGuard";
import { useAuthStore } from "@/stores/authStore";

// Helper to render a component inside a router
function renderWithRouter(ui: React.ReactElement, { route = "/protected" } = {}) {
  return render(
    <MemoryRouter initialEntries={[route]}>
      <Routes>
        <Route path="/login" element={<div data-testid="login-page">Login Page</div>} />
        <Route path="/vehicles" element={<div data-testid="vehicles-page">Vehicles Page</div>} />
        <Route path="/protected" element={ui} />
      </Routes>
    </MemoryRouter>
  );
}

describe("AuthGuard", () => {
  beforeEach(() => {
    // Reset auth store
    useAuthStore.setState({
      accessToken: null,
      refreshToken: null,
      role: null,
      email: null,
      userId: null,
      isAuthenticated: false,
    });
  });

  it("redirects unauthenticated users to /login", () => {
    useAuthStore.setState({ isAuthenticated: false });

    renderWithRouter(
      <AuthGuard>
        <div>Protected Content</div>
      </AuthGuard>
    );

    // Should see the login page, not protected content
    expect(screen.getByTestId("login-page")).toBeInTheDocument();
  });

  it("renders children for authenticated users", () => {
    useAuthStore.setState({
      accessToken: "valid-token",
      isAuthenticated: true,
      role: "user",
    });

    renderWithRouter(
      <AuthGuard>
        <div>Protected Content</div>
      </AuthGuard>
    );

    expect(screen.getByText("Protected Content")).toBeInTheDocument();
  });
});

describe("AdminGuard", () => {
  beforeEach(() => {
    useAuthStore.setState({
      accessToken: "valid-token",
      isAuthenticated: true,
      refreshToken: "refresh-token",
    });
  });

  it("renders children for admin users", () => {
    useAuthStore.setState({ role: "admin" });

    renderWithRouter(
      <AdminGuard>
        <div>Admin Content</div>
      </AdminGuard>
    );

    expect(screen.getByText("Admin Content")).toBeInTheDocument();
  });

  it("redirects non-admin users to /vehicles", () => {
    useAuthStore.setState({ role: "user" });

    renderWithRouter(
      <AdminGuard>
        <div>Admin Content</div>
      </AdminGuard>
    );

    // Should see vehicles page, not admin content
    expect(screen.getByTestId("vehicles-page")).toBeInTheDocument();
  });

  it("redirects users with null role to /vehicles", () => {
    useAuthStore.setState({ role: null });

    renderWithRouter(
      <AdminGuard>
        <div>Admin Content</div>
      </AdminGuard>
    );

    expect(screen.getByTestId("vehicles-page")).toBeInTheDocument();
  });
});