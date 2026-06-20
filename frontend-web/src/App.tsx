import { Outlet } from "react-router-dom";
import { RealtimeProvider } from "@/components/RealtimeProvider";

export default function App() {
  return (
    <RealtimeProvider>
      <div className="min-h-screen bg-slate-900 text-white">
        <Outlet />
      </div>
    </RealtimeProvider>
  );
}