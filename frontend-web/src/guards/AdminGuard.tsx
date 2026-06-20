// Admin-only route guard — redirects non-admins to /vehicles

import { Navigate } from "react-router-dom";
import { useAuthStore } from "@/stores/authStore";
import type { ReactNode } from "react";

interface AdminGuardProps {
  children: ReactNode;
}

export function AdminGuard({ children }: AdminGuardProps) {
  const role = useAuthStore((state) => state.role);

  if (role !== "admin") {
    return <Navigate to="/vehicles" replace />;
  }

  return <>{children}</>;
}