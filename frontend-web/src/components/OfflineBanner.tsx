import { useOnline } from "@/hooks/useOnline";
import { cn } from "@/utils/cn";

export function OfflineBanner() {
  const online = useOnline();

  if (online) return null;

  return (
    <div
      role="alert"
      className={cn(
        "sticky top-0 z-50 bg-amber-600 px-4 py-2 text-center text-sm font-medium text-white"
      )}
    >
      You are offline. Showing cached data.
    </div>
  );
}