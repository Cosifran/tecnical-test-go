import { useEffect } from "react";
import { useToastStore } from "@/stores/toastStore";
import { cn } from "@/utils/cn";

const AUTO_DISMISS_MS = 5000;

interface ToastProps {
  id: string;
  type: "low_fuel" | "info" | "error";
  message: string;
  vehicleId?: string;
  estimatedMinutes?: number;
  createdAt: number;
  onDismiss: (id: string) => void;
}

export function Toast({
  id,
  type,
  message,
  estimatedMinutes,
  onDismiss,
}: ToastProps) {
  useEffect(() => {
    const timer = setTimeout(() => {
      onDismiss(id);
    }, AUTO_DISMISS_MS);
    return () => clearTimeout(timer);
  }, [id, onDismiss]);

  const bgColor = type === "low_fuel" ? "bg-red-600" : type === "error" ? "bg-red-800" : "bg-blue-600";

  return (
    <div
      role="alert"
      className={cn(
        "flex items-center justify-between rounded-lg px-4 py-3 text-white shadow-lg",
        bgColor
      )}
    >
      <div className="flex flex-col gap-0.5">
        <span className="text-sm font-semibold">{message}</span>
        {estimatedMinutes !== undefined && (
          <span className="text-xs opacity-80">
            ~{estimatedMinutes} minutes of fuel remaining
          </span>
        )}
      </div>
      <button
        type="button"
        onClick={() => onDismiss(id)}
        className="ml-3 rounded-full p-1 text-white hover:bg-white/20 transition-colors"
        aria-label="Dismiss"
      >
        <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  );
}

export function ToastContainer() {
  const toasts = useToastStore((s) => s.toasts);
  const removeToast = useToastStore((s) => s.removeToast);

  if (toasts.length === 0) return null;

  return (
    <div
      data-testid="toast-container"
      className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm"
    >
      {toasts.map((toast) => (
        <Toast
          key={toast.id}
          id={toast.id}
          type={toast.type}
          message={toast.message}
          vehicleId={toast.vehicleId}
          estimatedMinutes={toast.estimatedMinutes}
          createdAt={toast.createdAt}
          onDismiss={removeToast}
        />
      ))}
    </div>
  );
}