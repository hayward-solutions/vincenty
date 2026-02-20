"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { api, ApiError } from "@/lib/api";
import type { AuthResponse, User } from "@/types/api";

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  isAdmin: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Check for existing session on mount
  useEffect(() => {
    const token = localStorage.getItem("access_token");
    if (!token) {
      setIsLoading(false);
      return;
    }

    api
      .get<User>("/api/v1/users/me")
      .then(setUser)
      .catch(() => {
        api.clearTokens();
      })
      .finally(() => setIsLoading(false));
  }, []);

  const login = useCallback(async (username: string, password: string) => {
    const data = await api.post<AuthResponse>("/api/v1/auth/login", {
      username,
      password,
    });
    api.setTokens(data.access_token, data.refresh_token);
    setUser(data.user);
  }, []);

  const refreshUser = useCallback(async () => {
    try {
      const updated = await api.get<User>("/api/v1/users/me");
      setUser(updated);
    } catch {
      // Silently ignore — user state remains stale until next reload
    }
  }, []);

  const logout = useCallback(async () => {
    const refreshToken = localStorage.getItem("refresh_token");
    try {
      if (refreshToken) {
        await api.post("/api/v1/auth/logout", {
          refresh_token: refreshToken,
        });
      }
    } catch (err) {
      // Ignore errors during logout
      if (!(err instanceof ApiError)) throw err;
    } finally {
      api.clearTokens();
      setUser(null);
    }
  }, []);

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading,
        isAuthenticated: !!user,
        isAdmin: user?.is_admin ?? false,
        login,
        logout,
        refreshUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
