import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, RouterProvider, Navigate } from "react-router-dom";
import App from "./App";
import { AuthGuard } from "./guards/AuthGuard";
import { AdminGuard } from "./guards/AdminGuard";
import { Layout } from "./components/Layout";
import { LoginPage } from "./pages/LoginPage";
import { VehicleListPage } from "./pages/VehicleListPage";
import { VehicleDetailPage } from "./pages/VehicleDetailPage";
import { AlertsPage } from "./pages/AlertsPage";
import "maplibre-gl/dist/maplibre-gl.css";
import "./styles/globals.css";

const router = createBrowserRouter([
  {
    path: "/login",
    element: <LoginPage />,
  },
  {
    path: "/",
    element: <App />,
    children: [
      {
        element: <Layout />,
        children: [
          {
            index: true,
            element: <Navigate to="/vehicles" replace />,
          },
          {
            path: "vehicles",
            element: (
              <AuthGuard>
                <VehicleListPage />
              </AuthGuard>
            ),
          },
          {
            path: "vehicles/:id",
            element: (
              <AuthGuard>
                <VehicleDetailPage />
              </AuthGuard>
            ),
          },
          {
            path: "alerts",
            element: (
              <AdminGuard>
                <AlertsPage />
              </AdminGuard>
            ),
          },
        ],
      },
    ],
  },
]);

const root = document.getElementById("root");

if (root) {
  createRoot(root).render(
    <StrictMode>
      <RouterProvider router={router} />
    </StrictMode>
  );
}