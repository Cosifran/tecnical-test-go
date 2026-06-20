import { useAuthStore } from "@/stores/authStore";

export function useAuth() {
  const accessToken = useAuthStore((state) => state.accessToken);
  const refreshToken = useAuthStore((state) => state.refreshToken);
  const role = useAuthStore((state) => state.role);
  const email = useAuthStore((state) => state.email);
  const userId = useAuthStore((state) => state.userId);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const login = useAuthStore((state) => state.login);
  const logout = useAuthStore((state) => state.logout);
  const refresh = useAuthStore((state) => state.refresh);

  return {
    accessToken,
    refreshToken,
    role,
    email,
    userId,
    isAuthenticated,
    isAdmin: role === "admin",
    login,
    logout,
    refresh,
  };
}