import { Outlet, NavLink, useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/authStore";
import { maskDeviceId } from "@/utils/masking";

export function Layout() {
  const email = useAuthStore((state) => state.email);
  const role = useAuthStore((state) => state.role);
  const logout = useAuthStore((state) => state.logout);
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate("/login");
  };

  const navLinkClass = ({ isActive }: { isActive: boolean }) =>
    `flex items-center gap-2 rounded px-3 py-2 text-sm font-medium transition-colors ${
      isActive
        ? "bg-blue-600 text-white"
        : "text-slate-300 hover:bg-slate-700 hover:text-white"
    }`;

  return (
    <div className="flex min-h-screen bg-slate-900">
      {/* Sidebar */}
      <aside className="flex w-64 flex-col border-r border-slate-700 bg-slate-800">
        <div className="border-b border-slate-700 px-4 py-4">
          <h1 className="text-lg font-bold text-white">Fleet Monitor</h1>
        </div>

        <nav className="flex flex-1 flex-col gap-1 px-2 py-4">
          <NavLink to="/vehicles" className={navLinkClass}>
            <span>🚗</span> Vehicles
          </NavLink>

          {role === "admin" && (
            <NavLink to="/alerts" className={navLinkClass}>
              <span>🔔</span> Alerts
            </NavLink>
          )}
        </nav>

        {/* User info + logout */}
        <div className="border-t border-slate-700 px-4 py-3">
          <div className="mb-2 text-xs text-slate-400">
            <div className="truncate">{email}</div>
            <div className="text-slate-500">
              Role: <span className="font-medium text-slate-300">{role}</span>
            </div>
          </div>
          <button
            onClick={handleLogout}
            className="w-full rounded border border-slate-600 px-3 py-1.5 text-sm text-slate-300 hover:bg-slate-700 hover:text-white"
          >
            Logout
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  );
}