// WebSocket singleton — exponential backoff reconnect (1s→2s→4s→...→30s max).

import { useWStore } from "@/stores/wsStore";
import type { WsMessage } from "@/types/ws";

type MessageHandler = (msg: WsMessage) => void;

const INITIAL_RECONNECT_DELAY_MS = 1000;
const MAX_RECONNECT_DELAY_MS = 30_000;

export class WebSocketManager {
  private static instance: WebSocketManager | null = null;

  private ws: WebSocket | null = null;
  private url: string = "";
  private token: string = "";
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectAttempt: number = 0;
  private shouldReconnect: boolean = false;
  private subscribers: Map<string, Set<MessageHandler>> = new Map();

  private constructor() {}

  static getInstance(): WebSocketManager {
    if (!WebSocketManager.instance) {
      WebSocketManager.instance = new WebSocketManager();
    }
    return WebSocketManager.instance;
  }

  /** Connect with access token. Closes any previous connection first. */
  connect(url: string, token: string): void {
    this.url = url;
    this.token = token;
    this.shouldReconnect = true;

    if (this.ws) {
      this.ws.onclose = null; // Prevent reconnect from old connection
      this.ws.close(1000, "Reconnecting");
    }

    this.clearReconnectTimer();

    const fullUrl = `${url}?token=${token}`;
    this.ws = new WebSocket(fullUrl);

    this.ws.onopen = () => {
      this.onOpen();
    };

    this.ws.onmessage = (event: MessageEvent) => {
      this.onMessage(event);
    };

    this.ws.onerror = () => {
      // Error is usually followed by close — handled in onclose
    };

    this.ws.onclose = (event: CloseEvent) => {
      this.onClose(event);
    };
  }

  /** Disconnect gracefully — no automatic reconnect. */
  disconnect(): void {
    this.shouldReconnect = false;
    this.clearReconnectTimer();
    if (this.ws) {
      this.ws.onclose = null; // Prevent reconnect on close
      this.ws.close(1000, "Disconnect");
      this.ws = null;
    }
    useWStore.getState().setConnected(false);
    useWStore.getState().setLastMessage(null);
  }

  /** Subscribe to messages of a given type. Returns unsubscribe function. */
  subscribe(type: string, handler: MessageHandler): () => void {
    if (!this.subscribers.has(type)) {
      this.subscribers.set(type, new Set());
    }
    this.subscribers.get(type)!.add(handler);

    return () => {
      const typeSubs = this.subscribers.get(type);
      if (typeSubs) {
        typeSubs.delete(handler);
        if (typeSubs.size === 0) {
          this.subscribers.delete(type);
        }
      }
    };
  }

  private onOpen(): void {
    this.reconnectAttempt = 0;
    useWStore.getState().setConnected(true);
    useWStore.getState().setReconnectAttempt(0);
  }

  private onMessage(event: MessageEvent): void {
    try {
      const data = JSON.parse(event.data) as WsMessage;
      useWStore.getState().setLastMessage(data);

      // Dispatch to subscribers by type
      const typeSubs = this.subscribers.get(data.type);
      if (typeSubs) {
        typeSubs.forEach((handler) => handler(data));
      }
    } catch {
      // Ignore malformed messages
    }
  }

  private onClose(event: CloseEvent): void {
    useWStore.getState().setConnected(false);

    // Graceful close (1000) or explicit disconnect — don't reconnect
    if (event.code === 1000 || !this.shouldReconnect) {
      return;
    }

    // Unexpected close — schedule reconnect with exponential backoff
    this.scheduleReconnect();
  }

  private scheduleReconnect(): void {
    this.clearReconnectTimer();

    const delay = Math.min(
      INITIAL_RECONNECT_DELAY_MS * Math.pow(2, this.reconnectAttempt),
      MAX_RECONNECT_DELAY_MS
    );

    this.reconnectAttempt++;
    useWStore.getState().setReconnectAttempt(this.reconnectAttempt);

    this.reconnectTimer = setTimeout(() => {
      this.connect(this.url, this.token);
    }, delay);
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }
}