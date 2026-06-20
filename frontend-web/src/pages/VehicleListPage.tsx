import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useShallow } from "zustand/react/shallow";
import { useFleetStore } from "@/stores/fleetStore";
import { useAuthStore } from "@/stores/authStore";
import { VehicleCard } from "@/components/VehicleCard";
import { LoadingSpinner } from "@/components/LoadingSpinner";

export function VehicleListPage() {
  const navigate = useNavigate();
  const role = useAuthStore((state) => state.role);

  const { vehicles, isLoading, error, isFetching } = useFleetStore(
    useShallow((state) => ({
      vehicles: state.vehicles ?? [],
      isLoading: state.isLoading,
      error: state.error,
      isFetching: state.isFetching,
    }))
  );

  const fetchVehicles = useFleetStore((state) => state.fetchVehicles);
  const clearError = useFleetStore((state) => state.clearError);

  useEffect(() => {
    // Fetch vehicles on mount, but don't block if we have cached data
    fetchVehicles();
  }, [fetchVehicles]);

  // Show loading ONLY on initial load (no cached data)
  if (isLoading && vehicles.length === 0) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <LoadingSpinner label="Loading vehicles..." />
      </div>
    );
  }

  if (error && vehicles.length === 0) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="mb-2 text-lg font-semibold text-red-400">Error</p>
          <p className="text-slate-400">{error}</p>
          <div className="mt-4 flex gap-2 justify-center">
            <button
              type="button"
              onClick={() => {
                clearError();
                fetchVehicles(true);
              }}
              className="rounded bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700"
            >
              Retry
            </button>
          </div>
        </div>
      </div>
    );
  }

  if (vehicles.length === 0) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="mb-2 text-lg font-semibold text-white">No vehicles found</p>
          <p className="text-sm text-slate-400">
            No vehicles are currently registered in the fleet.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-white">Fleet Vehicles</h1>
        {isFetching && (
          <span className="text-sm text-slate-400">Refreshing...</span>
        )}
      </div>

      {error && (
        <div className="mb-4 rounded border border-red-500 bg-red-900/20 p-3 text-sm text-red-400">
          {error}
          <button
            type="button"
            onClick={() => {
              clearError();
              fetchVehicles(true);
            }}
            className="ml-2 underline hover:text-red-300"
          >
            Retry
          </button>
        </div>
      )}

      <div className="flex flex-col gap-3">
        {vehicles.map((vehicle) => (
          <VehicleCard
            key={vehicle.id}
            vehicle={vehicle}
            role={role ?? "user"}
            onSelect={(id) => navigate(`/vehicles/${id}`)}
          />
        ))}
      </div>
    </div>
  );
}
