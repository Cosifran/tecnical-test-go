import { useEffect } from "react";
import { useShallow } from "zustand/react/shallow";
import { useFleetStore } from "@/stores/fleetStore";
import { LoadingSpinner } from "@/components/LoadingSpinner";
import { cn } from "@/utils/cn";

const SEVERITY_STYLES = {
  critical: "bg-red-600 text-white",
  warning: "bg-amber-500 text-white",
  info: "bg-blue-600 text-white",
} as const;

function formatSeverity(severity: string): keyof typeof SEVERITY_STYLES | "default" {
  if (severity in SEVERITY_STYLES) return severity as keyof typeof SEVERITY_STYLES;
  return "default";
}

function formatDate(isoString: string): string {
  try {
    return new Date(isoString).toLocaleString();
  } catch {
    return isoString;
  }
}

export function AlertsPage() {
  const { alerts, alertLoading, alertError } = useFleetStore(
    useShallow((state) => ({
      alerts: state.alerts ?? [],
      alertLoading: state.alertLoading,
      alertError: state.alertError,
    }))
  );

  const fetchAlerts = useFleetStore((state) => state.fetchAlerts);
  const clearAlertError = useFleetStore((state) => state.clearAlertError);

  useEffect(() => {
    fetchAlerts();
  }, [fetchAlerts]);

  if (alertLoading && alerts.length === 0) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <LoadingSpinner label="Loading alerts..." />
      </div>
    );
  }

  if (alertError && alerts.length === 0) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="mb-2 text-lg font-semibold text-red-400">Error</p>
          <p className="text-slate-400">{alertError}</p>
          <div className="mt-4 flex gap-2 justify-center">
            <button
              type="button"
              onClick={() => {
                clearAlertError();
                fetchAlerts(true);
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

  if (alerts.length === 0) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="mb-2 text-lg font-semibold text-white">No alerts at this time</p>
          <p className="text-sm text-slate-400">
            Predictive alerts will appear here when vehicles need attention.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-white">Alerts</h1>
        <p className="mt-1 text-sm text-slate-400">
          Predictive alerts for fleet vehicles
        </p>
      </div>

      {alertError && (
        <div className="mb-4 rounded border border-red-500 bg-red-900/20 p-3 text-sm text-red-400">
          {alertError}
          <button
            type="button"
            onClick={() => {
              clearAlertError();
              fetchAlerts(true);
            }}
            className="ml-2 underline hover:text-red-300"
          >
            Retry
          </button>
        </div>
      )}

      <div className="flex flex-col gap-3">
        {alerts.map((alert) => {
          const severityKey = formatSeverity(alert.severity);
          const badgeClass = severityKey === "default"
            ? "bg-slate-600 text-white"
            : SEVERITY_STYLES[severityKey];

          return (
            <div
              key={alert.id}
              className="rounded-lg border border-slate-700 bg-slate-800 p-4"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <span className={cn("rounded px-2 py-0.5 text-xs font-medium", badgeClass)}>
                    {alert.severity}
                  </span>
                  <span className="font-medium text-white">{alert.type}</span>
                </div>
                <span className="text-xs text-slate-400">
                  {formatDate(alert.created_at)}
                </span>
              </div>

              <div className="mt-2 text-sm text-slate-300">
                <span className="text-slate-400">Vehicle: </span>
                <span className="font-mono">{alert.vehicle_id}</span>
              </div>

              {alert.details && Object.keys(alert.details).length > 0 && (
                <div className="mt-2 text-xs text-slate-400">
                  {Object.entries(alert.details).map(([key, value]) => (
                    <span key={key} className="mr-4">
                      {key}: {String(value)}
                    </span>
                  ))}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}