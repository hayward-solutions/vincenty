import React, { type ReactElement } from "react";
import { render, type RenderOptions } from "@testing-library/react";
import { AuthProvider } from "@/lib/auth-context";

// ---------------------------------------------------------------------------
// Mock WebSocket context
// ---------------------------------------------------------------------------

// A minimal mock of the WebSocket context for hooks that depend on it.
// This avoids needing the full WebSocketProvider (which tries to connect).

type MessageHandler = (type: string, payload: unknown) => void;

interface MockWebSocketContextType {
  connectionState: "connected" | "disconnected" | "connecting";
  deviceId: string | null;
  sendMessage: (type: string, payload: unknown) => void;
  subscribe: (handler: MessageHandler) => () => void;
}

const mockSubscribers = new Set<MessageHandler>();

export const mockWebSocket: MockWebSocketContextType = {
  connectionState: "connected",
  deviceId: "device-1",
  sendMessage: vi.fn(),
  subscribe: (handler: MessageHandler) => {
    mockSubscribers.add(handler);
    return () => {
      mockSubscribers.delete(handler);
    };
  },
};

/** Dispatch a mock WebSocket message to all subscribed handlers. */
export function dispatchWSMessage(type: string, payload: unknown) {
  mockSubscribers.forEach((handler) => handler(type, payload));
}

/** Clear all WS subscribers between tests. */
export function clearWSSubscribers() {
  mockSubscribers.clear();
}

// Mock the websocket-context module so hooks get our mock
vi.mock("@/lib/websocket-context", () => ({
  useWebSocket: () => mockWebSocket,
  WebSocketProvider: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));

// ---------------------------------------------------------------------------
// Mock auth context for hooks that use useAuth
// ---------------------------------------------------------------------------

// Many hooks import useAuth. We provide a simple mock that returns an
// authenticated user by default. Individual tests can override via
// vi.mock if needed.

const mockAuth = {
  user: {
    id: "user-1",
    username: "testuser",
    email: "test@example.com",
    display_name: "Test User",
    avatar_url: "",
    marker_icon: "default",
    marker_color: "#3b82f6",
    is_admin: false,
    is_active: true,
    mfa_enabled: false,
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
  isLoading: false,
  isAuthenticated: true,
  isAdmin: false,
  login: vi.fn(),
  completeMFALogin: vi.fn(),
  passkeyLogin: vi.fn(),
  logout: vi.fn(),
  refreshUser: vi.fn(),
};

vi.mock("@/lib/auth-context", () => ({
  useAuth: () => mockAuth,
  AuthProvider: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));

// ---------------------------------------------------------------------------
// Custom render
// ---------------------------------------------------------------------------

function AllProviders({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

/**
 * Custom render that wraps components in necessary providers.
 * Since auth and websocket are mocked at module level, we don't need
 * real providers here — just a passthrough wrapper.
 */
function customRender(
  ui: ReactElement,
  options?: Omit<RenderOptions, "wrapper">
) {
  return render(ui, { wrapper: AllProviders, ...options });
}

export { customRender as render };
export { mockAuth };
