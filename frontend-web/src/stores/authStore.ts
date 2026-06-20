// Auth store — JWT role/email extracted via manual base64 decode (NO external JWT lib).
// Persisted to localStorage for offline resilience.

import { create } from "zustand";
import { persist } from "zustand/middleware";
import { decodeJWT } from "@/utils/jwt";
import type { LoginResponse } from "@/types/api";
import { login as loginEndpoint, refreshToken as refreshEndpoint } from "@/api/endpoints";

export interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  role: "admin" | "user" | null;
  email: string | null;
  userId: string | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  refresh: () => Promise<boolean>;
  setTokens: (accessToken: string, refreshToken: string) => void;
}

function extractClaimsFromToken(token: string): {
  userId: string | null;
  email: string | null;
  role: "admin" | "user" | null;
} {
  const payload = decodeJWT(token);
  if (!payload) {
    return { userId: null, email: null, role: null };
  }
  return {
    userId: (payload.sub as string) ?? null,
    email: (payload.email as string) ?? null,
    role: (payload.role as "admin" | "user") ?? null,
  };
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      accessToken: null,
      refreshToken: null,
      role: null,
      email: null,
      userId: null,
      isAuthenticated: false,

      login: async (email: string, password: string) => {
        const response: LoginResponse = await loginEndpoint({ email, password });
        const claims = extractClaimsFromToken(response.access_token);
        set({
          accessToken: response.access_token,
          refreshToken: response.refresh_token,
          role: claims.role,
          email: claims.email,
          userId: claims.userId,
          isAuthenticated: true,
        });
      },

      logout: () => {
        set({
          accessToken: null,
          refreshToken: null,
          role: null,
          email: null,
          userId: null,
          isAuthenticated: false,
        });
      },

      refresh: async (): Promise<boolean> => {
        const currentRefreshToken = get().refreshToken;
        if (!currentRefreshToken) {
          return false;
        }

        try {
          const response: LoginResponse = await refreshEndpoint(currentRefreshToken);
          const claims = extractClaimsFromToken(response.access_token);
          set({
            accessToken: response.access_token,
            refreshToken: response.refresh_token,
            role: claims.role,
            email: claims.email,
            userId: claims.userId,
            isAuthenticated: true,
          });
          return true;
        } catch {
          return false;
        }
      },

      setTokens: (accessToken: string, newRefreshToken: string) => {
        const claims = extractClaimsFromToken(accessToken);
        set({
          accessToken,
          refreshToken: newRefreshToken,
          role: claims.role,
          email: claims.email,
          userId: claims.userId,
          isAuthenticated: true,
        });
      },
    }),
    {
      name: "auth-storage",
      partialize: (state) => ({
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
        role: state.role,
        email: state.email,
        userId: state.userId,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);