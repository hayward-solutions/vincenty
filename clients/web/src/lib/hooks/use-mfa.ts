"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import type {
  MFAMethod,
  TOTPSetupResponse,
  TOTPVerifyResponse,
  RecoveryCodesResponse,
  WebAuthnRegisterResponse,
  ServerSettings,
  AuthResponse,
  PasskeyBeginResponse,
} from "@/types/api";

// ---------------------------------------------------------------------------
// MFA method listing
// ---------------------------------------------------------------------------

export function useMFAMethods() {
  const [methods, setMethods] = useState<MFAMethod[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const fetch = useCallback(async () => {
    setIsLoading(true);
    try {
      const data = await api.get<MFAMethod[]>("/api/v1/users/me/mfa/methods");
      setMethods(data);
    } catch {
      setMethods([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetch();
  }, [fetch]);

  return { methods, isLoading, refetch: fetch };
}

// ---------------------------------------------------------------------------
// TOTP setup
// ---------------------------------------------------------------------------

export function useTOTPSetup() {
  const [isLoading, setIsLoading] = useState(false);

  const beginSetup = useCallback(async (name: string) => {
    setIsLoading(true);
    try {
      return await api.post<TOTPSetupResponse>(
        "/api/v1/users/me/mfa/totp/setup",
        { name }
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  const verifySetup = useCallback(
    async (methodId: string, code: string) => {
      setIsLoading(true);
      try {
        return await api.post<TOTPVerifyResponse>(
          "/api/v1/users/me/mfa/totp/verify",
          { method_id: methodId, code }
        );
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  return { beginSetup, verifySetup, isLoading };
}

// ---------------------------------------------------------------------------
// WebAuthn registration
// ---------------------------------------------------------------------------

/** Server response from go-webauthn BeginRegistration (wraps options in `publicKey`). */
export interface CredentialCreationResponse {
  publicKey: PublicKeyCredentialCreationOptions;
}

export function useWebAuthnRegister() {
  const [isLoading, setIsLoading] = useState(false);

  const beginRegister = useCallback(async (name: string) => {
    setIsLoading(true);
    try {
      const resp = await api.post<CredentialCreationResponse>(
        "/api/v1/users/me/mfa/webauthn/register/begin",
        { name }
      );
      return resp;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const finishRegister = useCallback(
    async (credential: Credential) => {
      setIsLoading(true);
      try {
        return await api.post<WebAuthnRegisterResponse>(
          "/api/v1/users/me/mfa/webauthn/register/finish",
          credential
        );
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  return { beginRegister, finishRegister, isLoading };
}

// ---------------------------------------------------------------------------
// Delete MFA method
// ---------------------------------------------------------------------------

export function useDeleteMFAMethod() {
  const [isLoading, setIsLoading] = useState(false);

  const deleteMethod = useCallback(async (methodId: string) => {
    setIsLoading(true);
    try {
      await api.delete(`/api/v1/users/me/mfa/methods/${methodId}`);
    } finally {
      setIsLoading(false);
    }
  }, []);

  return { deleteMethod, isLoading };
}

// ---------------------------------------------------------------------------
// Toggle passwordless
// ---------------------------------------------------------------------------

export function useTogglePasswordless() {
  const [isLoading, setIsLoading] = useState(false);

  const toggle = useCallback(
    async (credentialId: string, enabled: boolean) => {
      setIsLoading(true);
      try {
        await api.put(
          `/api/v1/users/me/mfa/webauthn/${credentialId}/passwordless`,
          { passwordless_enabled: enabled }
        );
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  return { toggle, isLoading };
}

// ---------------------------------------------------------------------------
// Recovery codes
// ---------------------------------------------------------------------------

export function useRecoveryCodes() {
  const [isLoading, setIsLoading] = useState(false);

  const regenerate = useCallback(async () => {
    setIsLoading(true);
    try {
      return await api.post<RecoveryCodesResponse>(
        "/api/v1/users/me/mfa/recovery-codes"
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  return { regenerate, isLoading };
}

// ---------------------------------------------------------------------------
// MFA login challenge
// ---------------------------------------------------------------------------

export function useMFAChallenge() {
  const [isLoading, setIsLoading] = useState(false);

  const verifyTOTP = useCallback(
    async (mfaToken: string, code: string) => {
      setIsLoading(true);
      try {
        return await api.post<AuthResponse>("/api/v1/auth/mfa/totp", {
          mfa_token: mfaToken,
          code,
        });
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  const verifyRecovery = useCallback(
    async (mfaToken: string, code: string) => {
      setIsLoading(true);
      try {
        return await api.post<AuthResponse>("/api/v1/auth/mfa/recovery", {
          mfa_token: mfaToken,
          code,
        });
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  const beginWebAuthn = useCallback(async (mfaToken: string) => {
    setIsLoading(true);
    try {
      return await api.post<{ options: unknown; mfa_token: string }>(
        "/api/v1/auth/mfa/webauthn/begin",
        { mfa_token: mfaToken }
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  const finishWebAuthn = useCallback(
    async (mfaToken: string, assertion: unknown) => {
      setIsLoading(true);
      try {
        return await api.post<AuthResponse>(
          "/api/v1/auth/mfa/webauthn/finish",
          { mfa_token: mfaToken, ...(assertion as object) }
        );
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  return { verifyTOTP, verifyRecovery, beginWebAuthn, finishWebAuthn, isLoading };
}

// ---------------------------------------------------------------------------
// Passkey login
// ---------------------------------------------------------------------------

export function usePasskeyLogin() {
  const [isLoading, setIsLoading] = useState(false);

  const beginPasskey = useCallback(async () => {
    setIsLoading(true);
    try {
      return await api.post<PasskeyBeginResponse>(
        "/api/v1/auth/passkey/begin"
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  const finishPasskey = useCallback(
    async (sessionId: string, assertion: unknown) => {
      setIsLoading(true);
      try {
        return await api.post<AuthResponse>("/api/v1/auth/passkey/finish", {
          session_id: sessionId,
          ...(assertion as object),
        });
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  return { beginPasskey, finishPasskey, isLoading };
}

// ---------------------------------------------------------------------------
// Admin: server settings
// ---------------------------------------------------------------------------

export function useServerSettings() {
  const [settings, setSettings] = useState<ServerSettings | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const fetch = useCallback(async () => {
    setIsLoading(true);
    try {
      const data = await api.get<ServerSettings>("/api/v1/server/settings");
      setSettings(data);
    } catch {
      // May not be admin
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetch();
  }, [fetch]);

  const update = useCallback(
    async (updates: Partial<ServerSettings>) => {
      setIsLoading(true);
      try {
        const data = await api.put<ServerSettings>(
          "/api/v1/server/settings",
          updates
        );
        setSettings(data);
        return data;
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  return { settings, isLoading, update, refetch: fetch };
}

// ---------------------------------------------------------------------------
// Admin: reset user MFA
// ---------------------------------------------------------------------------

export function useAdminResetMFA() {
  const [isLoading, setIsLoading] = useState(false);

  const resetMFA = useCallback(async (userId: string) => {
    setIsLoading(true);
    try {
      await api.delete(`/api/v1/users/${userId}/mfa`);
    } finally {
      setIsLoading(false);
    }
  }, []);

  return { resetMFA, isLoading };
}
