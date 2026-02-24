import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/msw-server";
import "@/test/test-utils";

import {
  useMFAMethods,
  useTOTPSetup,
  useWebAuthnRegister,
  useDeleteMFAMethod,
  useTogglePasswordless,
  useRecoveryCodes,
  useMFAChallenge,
  usePasskeyLogin,
  useServerSettings,
  useAdminResetMFA,
} from "@/lib/hooks/use-mfa";

beforeEach(() => {
  localStorage.setItem("access_token", "test-token");
});

describe("useMFAMethods", () => {
  it("fetches MFA methods on mount", async () => {
    const { result } = renderHook(() => useMFAMethods());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.methods).toHaveLength(1);
    expect(result.current.methods[0].type).toBe("totp");
  });

  it("returns empty array on error", async () => {
    server.use(
      http.get("/api/v1/users/me/mfa/methods", () => {
        return HttpResponse.json({ error: { message: "error" } }, { status: 500 });
      })
    );

    const { result } = renderHook(() => useMFAMethods());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.methods).toEqual([]);
  });
});

describe("useTOTPSetup", () => {
  it("begins TOTP setup", async () => {
    const { result } = renderHook(() => useTOTPSetup());

    let setup;
    await act(async () => {
      setup = await result.current.beginSetup("My Authenticator");
    });

    expect(setup).toMatchObject({ method_id: "mfa-method-1", secret: "JBSWY3DPEHPK3PXP" });
    expect(result.current.isLoading).toBe(false);
  });

  it("verifies TOTP", async () => {
    const { result } = renderHook(() => useTOTPSetup());

    let verified;
    await act(async () => {
      verified = await result.current.verifySetup("mfa-method-1", "123456");
    });

    expect(verified).toMatchObject({ verified: true });
    expect(verified?.recovery_codes).toHaveLength(3);
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useWebAuthnRegister", () => {
  it("begins WebAuthn registration", async () => {
    const { result } = renderHook(() => useWebAuthnRegister());

    let resp;
    await act(async () => {
      resp = await result.current.beginRegister("Security Key");
    });

    expect(resp).toHaveProperty("publicKey");
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useDeleteMFAMethod", () => {
  it("deletes an MFA method", async () => {
    const { result } = renderHook(() => useDeleteMFAMethod());

    await act(async () => {
      await result.current.deleteMethod("mfa-method-1");
    });

    expect(result.current.isLoading).toBe(false);
  });
});

describe("useTogglePasswordless", () => {
  it("toggles passwordless flag", async () => {
    const { result } = renderHook(() => useTogglePasswordless());

    await act(async () => {
      await result.current.toggle("cred-1", true);
    });

    expect(result.current.isLoading).toBe(false);
  });
});

describe("useRecoveryCodes", () => {
  it("regenerates recovery codes", async () => {
    const { result } = renderHook(() => useRecoveryCodes());

    let codes;
    await act(async () => {
      codes = await result.current.regenerate();
    });

    expect(codes?.codes).toHaveLength(4);
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useMFAChallenge", () => {
  it("verifies TOTP challenge", async () => {
    const { result } = renderHook(() => useMFAChallenge());

    let auth;
    await act(async () => {
      auth = await result.current.verifyTOTP("mfa-token-123", "654321");
    });

    expect(auth).toMatchObject({ access_token: "test-access-token" });
    expect(result.current.isLoading).toBe(false);
  });

  it("verifies recovery code", async () => {
    const { result } = renderHook(() => useMFAChallenge());

    let auth;
    await act(async () => {
      auth = await result.current.verifyRecovery("mfa-token-123", "aaaa-bbbb");
    });

    expect(auth).toMatchObject({ access_token: "test-access-token" });
  });

  it("begins WebAuthn MFA challenge", async () => {
    const { result } = renderHook(() => useMFAChallenge());

    let resp;
    await act(async () => {
      resp = await result.current.beginWebAuthn("mfa-token-123");
    });

    expect(resp).toHaveProperty("options");
    expect(resp).toHaveProperty("mfa_token");
  });

  it("finishes WebAuthn MFA challenge", async () => {
    const { result } = renderHook(() => useMFAChallenge());

    let auth;
    await act(async () => {
      auth = await result.current.finishWebAuthn("mfa-token-123", { assertion: "data" });
    });

    expect(auth).toMatchObject({ access_token: "test-access-token" });
  });
});

describe("usePasskeyLogin", () => {
  it("begins passkey login", async () => {
    const { result } = renderHook(() => usePasskeyLogin());

    let resp;
    await act(async () => {
      resp = await result.current.beginPasskey();
    });

    expect(resp).toHaveProperty("session_id", "session-1");
    expect(result.current.isLoading).toBe(false);
  });

  it("finishes passkey login", async () => {
    const { result } = renderHook(() => usePasskeyLogin());

    let auth;
    await act(async () => {
      auth = await result.current.finishPasskey("session-1", { id: "cred-1" });
    });

    expect(auth).toMatchObject({ access_token: "test-access-token" });
  });
});

describe("useServerSettings", () => {
  it("fetches server settings on mount", async () => {
    const { result } = renderHook(() => useServerSettings());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.settings).toMatchObject({ mfa_required: false });
  });

  it("updates server settings", async () => {
    const { result } = renderHook(() => useServerSettings());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    let updated;
    await act(async () => {
      updated = await result.current.update({ mfa_required: true });
    });

    expect(updated).toMatchObject({ mfa_required: true });
  });
});

describe("useAdminResetMFA", () => {
  it("resets user MFA", async () => {
    const { result } = renderHook(() => useAdminResetMFA());

    await act(async () => {
      await result.current.resetMFA("user-1");
    });

    expect(result.current.isLoading).toBe(false);
  });
});
