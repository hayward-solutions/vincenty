"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { api, ApiError } from "@/lib/api";
import type {
  AuthResponse,
  LoginResult,
  MFAChallengeResponse,
  PasskeyBeginResponse,
  User,
} from "@/types/api";
import { isMFAChallengeResponse } from "@/types/api";

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  isAdmin: boolean;
  /** Login with username/password. May return MFAChallengeResponse if MFA is enabled. */
  login: (
    username: string,
    password: string
  ) => Promise<LoginResult | undefined>;
  /** Complete MFA login after successful verification. */
  completeMFALogin: (resp: AuthResponse) => void;
  /** Passwordless passkey login. */
  passkeyLogin: () => Promise<void>;
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

  const login = useCallback(
    async (
      username: string,
      password: string
    ): Promise<LoginResult | undefined> => {
      const data = await api.post<LoginResult>("/api/v1/auth/login", {
        username,
        password,
      });

      // Check if MFA is required
      if (isMFAChallengeResponse(data)) {
        return data;
      }

      // Normal login success
      const authResp = data as AuthResponse;
      api.setTokens(authResp.access_token, authResp.refresh_token);
      setUser(authResp.user);
      return authResp;
    },
    []
  );

  const completeMFALogin = useCallback((resp: AuthResponse) => {
    api.setTokens(resp.access_token, resp.refresh_token);
    setUser(resp.user);
  }, []);

  const passkeyLogin = useCallback(async () => {
    // Begin passkey assertion
    const beginResp = await api.post<PasskeyBeginResponse>(
      "/api/v1/auth/passkey/begin"
    );

    // Server returns { options: { publicKey: { ... } }, session_id }
    // Extract .publicKey and convert base64url fields to ArrayBuffers.
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = (beginResp.options as any).publicKey as Record<string, unknown>;
    const publicKeyOptions: PublicKeyCredentialRequestOptions = {
      ...raw,
      challenge: base64URLToBuffer(raw.challenge as string),
      allowCredentials: (
        (raw.allowCredentials as Array<Record<string, unknown>>) ?? []
      ).map((c) => ({
        ...c,
        id: base64URLToBuffer(c.id as string),
      })) as PublicKeyCredentialDescriptor[],
    } as PublicKeyCredentialRequestOptions;

    // Call browser WebAuthn API for discoverable credential
    const assertion = await navigator.credentials.get({
      publicKey: publicKeyOptions,
    });

    if (!assertion) {
      throw new Error("No assertion returned");
    }

    const pkCred = assertion as PublicKeyCredential;
    const assertionResp = pkCred.response as AuthenticatorAssertionResponse;

    const authResp = await api.post<AuthResponse>(
      "/api/v1/auth/passkey/finish",
      {
        session_id: beginResp.session_id,
        id: pkCred.id,
        rawId: bufferToBase64URL(pkCred.rawId),
        type: pkCred.type,
        response: {
          authenticatorData: bufferToBase64URL(
            assertionResp.authenticatorData
          ),
          clientDataJSON: bufferToBase64URL(assertionResp.clientDataJSON),
          signature: bufferToBase64URL(assertionResp.signature),
          userHandle: assertionResp.userHandle
            ? bufferToBase64URL(assertionResp.userHandle)
            : null,
        },
      }
    );

    api.setTokens(authResp.access_token, authResp.refresh_token);
    setUser(authResp.user);
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
        completeMFALogin,
        passkeyLogin,
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

function base64URLToBuffer(base64url: string): ArrayBuffer {
  const base64 = base64url.replace(/-/g, "+").replace(/_/g, "/");
  const pad = base64.length % 4;
  const padded = pad ? base64 + "=".repeat(4 - pad) : base64;
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}

function bufferToBase64URL(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let str = "";
  for (const b of bytes) {
    str += String.fromCharCode(b);
  }
  return btoa(str).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}
