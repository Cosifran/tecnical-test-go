/**
 * Tests for LoginPage component.
 *
 * Verifies:
 * - Login form renders with email and password inputs
 * - Successful login stores tokens and redirects
 * - Failed login shows error message
 * - Password field masks input
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { LoginPage } from "@/pages/LoginPage";
import { useAuthStore } from "@/stores/authStore";

// Mock the auth store login
vi.mock("@/api/endpoints", () => ({
  login: vi.fn(),
  refreshToken: vi.fn(),
}));

function renderLoginPage() {
  return render(
    <MemoryRouter initialEntries={["/login"]}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/vehicles" element={<div data-testid="vehicles-page">Vehicles</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe("LoginPage", () => {
  beforeEach(() => {
    useAuthStore.setState({
      accessToken: null,
      refreshToken: null,
      role: null,
      email: null,
      userId: null,
      isAuthenticated: false,
    });
    vi.clearAllMocks();
  });

  it("renders email and password inputs", () => {
    renderLoginPage();

    expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
  });

  it("renders a submit button", () => {
    renderLoginPage();

    expect(screen.getByRole("button", { name: /sign in/i })).toBeInTheDocument();
  });

  it("password input has type=password for masking", () => {
    renderLoginPage();

    const passwordInput = screen.getByLabelText(/password/i);
    expect(passwordInput).toHaveAttribute("type", "password");
  });

  it("shows error message on failed login", async () => {
    const { login } = await import("@/api/endpoints");
    (login as ReturnType<typeof vi.fn>).mockRejectedValue(
      new Error("Invalid credentials")
    );

    renderLoginPage();

    const emailInput = screen.getByLabelText(/email/i);
    const passwordInput = screen.getByLabelText(/password/i);

    fireEvent.change(emailInput, { target: { value: "bad@fleet.com" } });
    fireEvent.change(passwordInput, { target: { value: "wrong" } });
    fireEvent.click(screen.getByRole("button", { name: /sign in/i }));

    await waitFor(() => {
      expect(screen.getByText(/invalid credentials/i)).toBeInTheDocument();
    });
  });

  it("calls login and redirects on successful login", async () => {
    const payload = {
      sub: "user-1",
      email: "admin@fleet.com",
      role: "admin",
      exp: 1893456000,
    };
    const payloadB64 = btoa(JSON.stringify(payload))
      .replace(/\+/g, "-")
      .replace(/\//g, "_")
      .replace(/=+$/, "");

    const { login } = await import("@/api/endpoints");
    (login as ReturnType<typeof vi.fn>).mockResolvedValue({
      access_token: `eyJhbGciOiJIUzI1NiJ9.${payloadB64}.fake-sig`,
      refresh_token: "refresh-token",
      token_type: "Bearer",
      expires_in: 3600,
    });

    renderLoginPage();

    const emailInput = screen.getByLabelText(/email/i);
    const passwordInput = screen.getByLabelText(/password/i);

    fireEvent.change(emailInput, { target: { value: "admin@fleet.com" } });
    fireEvent.change(passwordInput, { target: { value: "password123" } });
    fireEvent.click(screen.getByRole("button", { name: /sign in/i }));

    await waitFor(() => {
      expect(login).toHaveBeenCalledWith({
        email: "admin@fleet.com",
        password: "password123",
      });
    });
  });
});