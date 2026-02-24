import { describe, it, expect } from "vitest";
import { isMFAChallengeResponse } from "@/types/api";
import type { AuthResponse, MFAChallengeResponse } from "@/types/api";

describe("isMFAChallengeResponse", () => {
  it("returns true for MFA challenge response", () => {
    const challenge: MFAChallengeResponse = {
      mfa_required: true,
      mfa_token: "token-123",
      methods: ["totp", "recovery"],
    };
    expect(isMFAChallengeResponse(challenge)).toBe(true);
  });

  it("returns false for normal auth response", () => {
    const auth: AuthResponse = {
      access_token: "access-123",
      refresh_token: "refresh-456",
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
    };
    expect(isMFAChallengeResponse(auth)).toBe(false);
  });

  it("returns false when mfa_required is not true", () => {
    // Edge case: mfa_required exists but is false
    const data = {
      mfa_required: false,
      mfa_token: "token-123",
      methods: ["totp"],
    } as unknown as MFAChallengeResponse;
    expect(isMFAChallengeResponse(data)).toBe(false);
  });
});
