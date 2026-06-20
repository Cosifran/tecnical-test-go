import { useWebSocket } from "@/websocket/useWebSocket";
import { useOnline } from "@/hooks/useOnline";
import { OfflineBanner } from "@/components/OfflineBanner";
import { ToastContainer } from "@/components/Toast";

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  useOnline();
  useWebSocket();

  return (
    <>
      <OfflineBanner />
      <ToastContainer />
      {children}
    </>
  );
}