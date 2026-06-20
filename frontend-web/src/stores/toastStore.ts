// Toast store — ephemeral (not persisted). Admin-only low_fuel alerts shown here.

import { create } from "zustand";

export interface Toast {
  id: string;
  type: "low_fuel" | "info" | "error";
  message: string;
  vehicleId?: string;
  estimatedMinutes?: number;
  createdAt: number;
}

export interface ToastState {
  toasts: Toast[];
  addToast: (toast: Omit<Toast, "id" | "createdAt">) => void;
  removeToast: (id: string) => void;
  clearToasts: () => void;
}

let toastCounter = 0;

export const useToastStore = create<ToastState>()((set) => ({
  toasts: [],

  addToast: (toast) => {
    const id = `toast-${++toastCounter}`;
    set((state) => ({
      toasts: [...state.toasts, { ...toast, id, createdAt: Date.now() }],
    }));
  },

  removeToast: (id: string) => {
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    }));
  },

  clearToasts: () => {
    set({ toasts: [] });
  },
}));