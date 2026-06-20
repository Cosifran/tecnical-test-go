import { useEffect } from "react";
import { useParams, Link } from "react-router-dom";
import { useShallow } from "zustand/react/shallow";
import { useFleetStore } from "@/stores/fleetStore";
import { useAuthStore } from "@/stores/authStore";
import { maskDeviceId } from "@/utils/masking";
import { MapView } from "@/components/MapView";

export function VehicleDetailPage() {
  const { id } = useParams<{ id: string }>();
  const role = useAuthStore((state) => state.role);

  const { vehicle, history, liveGPS } = useFleetStore(
    useShallow((state) => ({
      vehicle: state.vehicles.find((v) => v.id === id),
      history: id ? state.history[id] : undefined,
      liveGPS: id ? state.liveGPS[id] : undefined,
    }))
  );

  const fetchHistory = useFleetStore((state) => state.fetchHistory);

  useEffect(() => {
    if (id && !history) {
      fetchHistory(id);
    }
  }, [id, history, fetchHistory]);

  if (!vehicle) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="mb-2 text-lg font-semibold text-white">Vehicle not found</p>
          <Link
            to="/vehicles"
            className="text-sm text-blue-400 hover:text-blue-300"
          >
            ← Back to vehicles
          </Link>
        </div>
      </div>
    );
  }

  const displayDeviceId = role === "admin" ? vehicle.device_id : maskDeviceId(vehicle.device_id);

  // Use live GPS from WebSocket, or fallback to history GPS
  const gpsPosition = liveGPS ?? (() => {
    if (!history) return null;
    const gpsReading = history
      .filter((p) => p.Type === "gps")
      .sort((a, b) => new Date(b.Timestamp).getTime() - new Date(a.Timestamp).getTime())[0];
    if (!gpsReading) return null;
    const val = gpsReading.Value as { lat: number; lng: number };
    return { lat: val.lat, lng: val.lng };
  })();

  console.log("History: ", history)

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6">
        <Link
          to="/vehicles"
          className="mb-2 inline-block text-sm text-blue-400 hover:text-blue-300"
        >
          ← Back to vehicles
        </Link>

        <h1 className="text-2xl font-bold text-white">{vehicle.name}</h1>

        <div className="mt-2 flex flex-col gap-1 text-sm text-slate-400">
          <span>Device ID: {displayDeviceId}</span>
          <span>Created: {new Date(vehicle.created_at).toLocaleString()}</span>
          {gpsPosition && (
            <span className="text-green-400">
              📍 Live: {gpsPosition.lat.toFixed(4)},{" "}
              {gpsPosition.lng.toFixed(4)}
            </span>
          )}
        </div>
      </div>

      {/* Map with LIVE position */}
      <div className="mb-6">
        {gpsPosition ? (
          <MapView
            center={[gpsPosition.lng, gpsPosition.lat]}
            markerLabel={vehicle.name.charAt(0)}
          />
        ) : (
          <div className="flex h-96 items-center justify-center rounded-lg border border-slate-700 bg-slate-800">
            <p className="text-sm text-slate-400">Waiting for GPS data...</p>
          </div>
        )}
      </div>

      {/* History */}
      <div className="rounded-lg border border-slate-700 bg-slate-800 p-4">
        <h2 className="mb-4 text-lg font-semibold text-white">
          Sensor History
        </h2>
        {history && history.length > 0 ? (
          <div className="flex flex-col gap-2 max-h-64 overflow-auto">
            {history.slice(0, 20).map((point, i) => (
              <div
                key={i}
                className="flex justify-between text-sm text-slate-300 border-b border-slate-700 pb-2"
              >
                <span>{new Date(point.Timestamp).toLocaleString()}</span>
                <div className="text-slate-400">
                  {point.Type === "gps" &&
                    `${point.Value.lat}, ${point.Value.lng}`}{" "}
                  {point.Type === "fuel" &&
                    `${point.Value.level} ${point.Value.unit}`}{" "}
                  {point.Type === "temperature" &&
                    `${point.Value.celsius} °C`}{" "}
                  <span>({point.Type})</span>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-slate-400">No history data available.</p>
        )}
      </div>
    </div>
  );
}
