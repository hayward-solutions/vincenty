"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { useAuth } from "@/lib/auth-context";
import { api } from "@/lib/api";
import type { Device, WSEnvelope } from "@/types/api";

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080";

type ConnectionState = "connecting" | "connected" | "disconnected";
type MessageHandler = (type: string, payload: unknown) => void;

interface WebSocketContextType {
  connectionState: ConnectionState;
  deviceId: string | null;
  sendMessage: (type: string, payload: unknown) => void;
  subscribe: (handler: MessageHandler) => () => void;
}

const WebSocketContext = createContext<WebSocketContextType | undefined>(
  undefined
);

// Maximum reconnect backoff in ms.
const MAX_BACKOFF = 30_000;

export function WebSocketProvider({ children }: { children: React.ReactNode }) {
  const { user, isAuthenticated } = useAuth();
  const [connectionState, setConnectionState] =
    useState<ConnectionState>("disconnected");
  const [deviceId, setDeviceId] = useState<string | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const handlersRef = useRef<Set<MessageHandler>>(new Set());
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const backoffRef = useRef(1000);
  const mountedRef = useRef(true);

  // -----------------------------------------------------------------------
  // Device auto-registration
  // -----------------------------------------------------------------------
  const ensureDevice = useCallback(async (): Promise<string | null> => {
    // Check localStorage first
    const stored = localStorage.getItem("device_id");
    if (stored) {
      setDeviceId(stored);
      return stored;
    }

    // Register a new web device
    try {
      const device = await api.post<Device>("/api/v1/users/me/devices", {
        name: "Web Browser",
        device_type: "web",
      });
      localStorage.setItem("device_id", device.id);
      setDeviceId(device.id);
      return device.id;
    } catch {
      console.error("Failed to register device");
      return null;
    }
  }, []);

  // -----------------------------------------------------------------------
  // Message dispatch
  // -----------------------------------------------------------------------
  const dispatch = useCallback((type: string, payload: unknown) => {
    handlersRef.current.forEach((handler) => {
      try {
        handler(type, payload);
      } catch (err) {
        console.error("WS handler error:", err);
      }
    });
  }, []);

  // -----------------------------------------------------------------------
  // Connect
  // -----------------------------------------------------------------------
  const connect = useCallback(
    async (devId: string) => {
      if (!mountedRef.current) return;

      const token = localStorage.getItem("access_token");
      if (!token) return;

      setConnectionState("connecting");

      const url = `${WS_URL}/api/v1/ws?token=${encodeURIComponent(token)}&device_id=${encodeURIComponent(devId)}`;

      try {
        const socket = new WebSocket(url);
        wsRef.current = socket;

        socket.onopen = () => {
          if (!mountedRef.current) return;
          console.log("[WS] Connected to", url);
          setConnectionState("connected");
          backoffRef.current = 1000; // reset backoff on success
        };

        socket.onmessage = (event) => {
          try {
            const envelope: WSEnvelope = JSON.parse(event.data);
            dispatch(envelope.type, envelope.payload);
          } catch {
            console.warn("Failed to parse WS message:", event.data);
          }
        };

        socket.onclose = () => {
          if (!mountedRef.current) return;
          wsRef.current = null;
          setConnectionState("disconnected");

          // Reconnect with backoff
          const delay = backoffRef.current;
          backoffRef.current = Math.min(delay * 2, MAX_BACKOFF);
          reconnectTimer.current = setTimeout(() => {
            if (mountedRef.current && localStorage.getItem("access_token")) {
              connect(devId);
            }
          }, delay);
        };

        socket.onerror = () => {
          // onclose will fire after this, triggering reconnect
          socket.close();
        };
      } catch {
        setConnectionState("disconnected");
      }
    },
    [dispatch]
  );

  // -----------------------------------------------------------------------
  // Lifecycle: connect on auth, disconnect on logout
  // -----------------------------------------------------------------------
  useEffect(() => {
    mountedRef.current = true;

    if (!isAuthenticated || !user) {
      // Clean disconnect
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current);
      }
      setConnectionState("disconnected");
      return;
    }

    // Connect
    (async () => {
      const devId = await ensureDevice();
      if (devId && mountedRef.current) {
        connect(devId);
      }
    })();

    return () => {
      mountedRef.current = false;
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [isAuthenticated, user, ensureDevice, connect]);

  // -----------------------------------------------------------------------
  // Public API
  // -----------------------------------------------------------------------
  const sendMessage = useCallback((type: string, payload: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type, payload }));
    }
  }, []);

  const subscribe = useCallback((handler: MessageHandler) => {
    handlersRef.current.add(handler);
    return () => {
      handlersRef.current.delete(handler);
    };
  }, []);

  return (
    <WebSocketContext.Provider
      value={{ connectionState, deviceId, sendMessage, subscribe }}
    >
      {children}
    </WebSocketContext.Provider>
  );
}

export function useWebSocket() {
  const context = useContext(WebSocketContext);
  if (context === undefined) {
    throw new Error("useWebSocket must be used within a WebSocketProvider");
  }
  return context;
}
