// WebSocket store — ephemeral (resets on reload; WS connections can't survive navigation).
// Toast notifications are handled separately in toastStore.

import { create } from "zustand";
import type { WsMessage } from "@/types/ws";

export interface WsState {
  connected: boolean;
  reconnectAttempt: number;
  lastMessage: WsMessage | null;
  setConnected: (value: boolean) => void;
  setReconnectAttempt: (value: number) => void;
  setLastMessage: (msg: WsMessage | null) => void;
  reset: () => void;
}

export const useWStore = create<WsState>()((set) => ({
  connected: false,
  reconnectAttempt: 0,
  lastMessage: null,

  setConnected: (value: boolean) => set({ connected: value }),
  setReconnectAttempt: (value: number) => set({ reconnectAttempt: value }),
  setLastMessage: (msg: WsMessage | null) => set({ lastMessage: msg }),
  reset: () =>
    set({
      connected: false,
      reconnectAttempt: 0,
      lastMessage: null,
    }),
}));