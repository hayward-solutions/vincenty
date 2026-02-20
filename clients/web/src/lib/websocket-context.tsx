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
import { DeviceEnrolmentDialog } from "@/components/devices/device-enrolment-dialog";
import type { Device, DeviceResolveResponse, WSEnvelope } from "@/types/api";

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

  // Enrolment prompt state: when non-null the dialog is shown and WS is paused.
  const [pendingEnrolment, setPendingEnrolment] = useState<Device[] | null>(
    null
  );

  const wsRef = useRef<WebSocket | null>(null);
  const handlersRef = useRef<Set<MessageHandler>>(new Set());
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const backoffRef = useRef(1000);
  const mountedRef = useRef(true);
  // Guard: only attempt device re-registration once per connect cycle.
  const retriedDeviceRef = useRef(false);

  // -----------------------------------------------------------------------
  // Device resolution
  // -----------------------------------------------------------------------

  /**
   * Attempt to resolve the current browser to an existing device.
   *
   * Returns a device ID when the device is immediately available (localStorage,
   * cookie/UA match, or first-login auto-create). Returns null when the user
   * needs to make a choice via the enrolment dialog — in that case
   * `pendingEnrolment` state is set and the caller should not connect the WS.
   */
  const ensureDevice = useCallback(async (): Promise<string | null> => {
    // 1. localStorage (fast path)
    const stored = localStorage.getItem("device_id");
    if (stored) {
      setDeviceId(stored);
      return stored;
    }

    // 2. Server-side resolve (cookie / UA heuristic)
    try {
      const result = await api.post<DeviceResolveResponse>(
        "/api/v1/users/me/devices/resolve"
      );

      if (result.matched && result.device) {
        // Cookie or UA matched — use it silently.
        localStorage.setItem("device_id", result.device.id);
        setDeviceId(result.device.id);
        return result.device.id;
      }

      const existing = result.existing_devices ?? [];

      if (existing.length === 0) {
        // 3. First login — no devices at all, auto-create silently.
        const device = await api.post<Device>("/api/v1/users/me/devices", {
          name: "Web Browser",
          device_type: "web",
        });
        localStorage.setItem("device_id", device.id);
        setDeviceId(device.id);
        return device.id;
      }

      // 4. User has existing devices but none matched — prompt.
      setPendingEnrolment(existing);
      return null;
    } catch {
      console.error("Failed to resolve device");
      return null;
    }
  }, []);

  /**
   * Called by the enrolment dialog once the user has made a choice.
   * Stores the device ID, clears the prompt, and triggers the WS connection.
   */
  const resolveEnrolment = useCallback(
    (id: string) => {
      localStorage.setItem("device_id", id);
      setDeviceId(id);
      setPendingEnrolment(null);
      // The useEffect watching deviceId will trigger connect.
    },
    []
  );

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

        let didOpen = false;

        socket.onopen = () => {
          if (!mountedRef.current) return;
          didOpen = true;
          retriedDeviceRef.current = false; // reset guard on success
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

          // If the server rejected the connection before it ever opened
          // (e.g. stale device_id returning 400), clear the stored device
          // and re-resolve once before falling back to normal backoff.
          if (!didOpen && !retriedDeviceRef.current) {
            retriedDeviceRef.current = true;
            localStorage.removeItem("device_id");
            console.warn("[WS] Connection rejected; re-resolving device");
            (async () => {
              const newDevId = await ensureDevice();
              if (newDevId && mountedRef.current) {
                connect(newDevId);
              }
            })();
            return;
          }

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
    [dispatch, ensureDevice]
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
      setPendingEnrolment(null);
      return;
    }

    // Connect
    (async () => {
      const devId = await ensureDevice();
      if (devId && mountedRef.current) {
        connect(devId);
      }
      // If devId is null the enrolment dialog is showing —
      // connection will happen via resolveEnrolment → the effect below.
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

  // When the enrolment dialog resolves, deviceId changes — connect if needed.
  useEffect(() => {
    if (
      deviceId &&
      isAuthenticated &&
      !wsRef.current &&
      connectionState === "disconnected" &&
      !pendingEnrolment
    ) {
      connect(deviceId);
    }
  }, [deviceId, isAuthenticated, connectionState, pendingEnrolment, connect]);

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
      {pendingEnrolment && (
        <DeviceEnrolmentDialog
          existingDevices={pendingEnrolment}
          onResolved={resolveEnrolment}
        />
      )}
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
