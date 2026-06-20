import { maskDeviceId } from "@/utils/masking";
import type { Vehicle } from "@/types/domain";

interface VehicleCardProps {
  vehicle: Vehicle;
  role: "admin" | "user";
  onSelect: (id: string) => void;
}

export function VehicleCard({ vehicle, role, onSelect }: VehicleCardProps) {
  const displayDeviceId = role === "admin" ? vehicle.device_id : maskDeviceId(vehicle.device_id);

  return (
    <button
      type="button"
      onClick={() => onSelect(vehicle.id)}
      className="flex w-full items-center gap-4 rounded-lg border border-slate-700 bg-slate-800 p-4 text-left transition-colors hover:border-blue-500 hover:bg-slate-750"
    >
      <div className="flex flex-1 flex-col gap-1">
        <span className="text-sm font-semibold text-white">{vehicle.name}</span>
        <span className="text-xs text-slate-400">{displayDeviceId}</span>
      </div>
      <div className="text-xs text-slate-500">
        {new Date(vehicle.created_at).toLocaleDateString()}
      </div>
    </button>
  );
}
