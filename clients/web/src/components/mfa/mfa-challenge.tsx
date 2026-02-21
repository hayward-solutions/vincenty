"use client";

import { useState } from "react";
import { useMFAChallenge } from "@/lib/hooks/use-mfa";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ApiError } from "@/lib/api";
import type { AuthResponse, MFAChallengeResponse } from "@/types/api";

interface MFAChallengeProps {
  challenge: MFAChallengeResponse;
  onSuccess: (resp: AuthResponse) => void;
  onCancel: () => void;
}

type Method = "totp" | "webauthn" | "recovery";

export function MFAChallenge({
  challenge,
  onSuccess,
  onCancel,
}: MFAChallengeProps) {
  const { verifyTOTP, verifyRecovery, beginWebAuthn, finishWebAuthn, isLoading } =
    useMFAChallenge();
  const [activeMethod, setActiveMethod] = useState<Method>(
    challenge.methods[0] as Method
  );
  const [code, setCode] = useState("");
  const [error, setError] = useState("");

  async function handleTOTPSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    try {
      const resp = await verifyTOTP(challenge.mfa_token, code);
      onSuccess(resp);
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : "Verification failed"
      );
    }
  }

  async function handleRecoverySubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    try {
      const resp = await verifyRecovery(challenge.mfa_token, code);
      onSuccess(resp);
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : "Invalid recovery code"
      );
    }
  }

  async function handleWebAuthn() {
    setError("");
    try {
      // Get assertion options from server
      // Server returns { options: { publicKey: { ... } }, mfa_token }
      const beginResp = await beginWebAuthn(challenge.mfa_token);

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

      // Call browser WebAuthn API
      const assertion = await navigator.credentials.get({
        publicKey: publicKeyOptions,
      });

      if (!assertion) {
        setError("No assertion returned");
        return;
      }

      const pkCred = assertion as PublicKeyCredential;
      const assertionResp =
        pkCred.response as AuthenticatorAssertionResponse;

      const resp = await finishWebAuthn(challenge.mfa_token, {
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
      });

      onSuccess(resp);
    } catch (err) {
      if (err instanceof DOMException && err.name === "NotAllowedError") {
        setError("Authentication was cancelled");
      } else {
        setError(
          err instanceof ApiError ? err.message : "WebAuthn verification failed"
        );
      }
    }
  }

  return (
    <Card className="w-full max-w-sm">
      <CardHeader>
        <CardTitle className="text-center text-lg">
          Two-Factor Authentication
        </CardTitle>
        <p className="text-center text-sm text-muted-foreground">
          Verify your identity to continue
        </p>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Method selector */}
        {challenge.methods.length > 1 && (
          <div className="flex gap-1 rounded-md border p-1">
            {challenge.methods.includes("totp") && (
              <Button
                variant={activeMethod === "totp" ? "secondary" : "ghost"}
                size="sm"
                className="flex-1 text-xs"
                onClick={() => {
                  setActiveMethod("totp");
                  setCode("");
                  setError("");
                }}
              >
                Authenticator
              </Button>
            )}
            {challenge.methods.includes("webauthn") && (
              <Button
                variant={activeMethod === "webauthn" ? "secondary" : "ghost"}
                size="sm"
                className="flex-1 text-xs"
                onClick={() => {
                  setActiveMethod("webauthn");
                  setCode("");
                  setError("");
                }}
              >
                Security Key
              </Button>
            )}
            {challenge.methods.includes("recovery") && (
              <Button
                variant={activeMethod === "recovery" ? "secondary" : "ghost"}
                size="sm"
                className="flex-1 text-xs"
                onClick={() => {
                  setActiveMethod("recovery");
                  setCode("");
                  setError("");
                }}
              >
                Recovery
              </Button>
            )}
          </div>
        )}

        {error && (
          <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
            {error}
          </div>
        )}

        {/* TOTP form */}
        {activeMethod === "totp" && (
          <form onSubmit={handleTOTPSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="mfa-totp">Authentication Code</Label>
              <Input
                id="mfa-totp"
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                placeholder="000000"
                maxLength={6}
                className="text-center text-lg font-mono tracking-widest"
                autoFocus
                required
              />
              <p className="text-xs text-muted-foreground">
                Enter the code from your authenticator app.
              </p>
            </div>
            <Button
              type="submit"
              className="w-full"
              disabled={isLoading || code.length !== 6}
            >
              {isLoading ? "Verifying..." : "Verify"}
            </Button>
          </form>
        )}

        {/* WebAuthn */}
        {activeMethod === "webauthn" && (
          <div className="space-y-4 text-center">
            <p className="text-sm text-muted-foreground">
              Use your security key to verify your identity.
            </p>
            <Button
              className="w-full"
              onClick={handleWebAuthn}
              disabled={isLoading}
            >
              {isLoading ? "Waiting..." : "Use Security Key"}
            </Button>
          </div>
        )}

        {/* Recovery code */}
        {activeMethod === "recovery" && (
          <form onSubmit={handleRecoverySubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="mfa-recovery">Recovery Code</Label>
              <Input
                id="mfa-recovery"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                placeholder="xxxx-xxxx"
                className="text-center font-mono"
                autoFocus
                required
              />
              <p className="text-xs text-muted-foreground">
                Enter one of your unused recovery codes.
              </p>
            </div>
            <Button
              type="submit"
              className="w-full"
              disabled={isLoading || !code.trim()}
            >
              {isLoading ? "Verifying..." : "Verify"}
            </Button>
          </form>
        )}

        <div className="text-center">
          <Button variant="link" size="sm" onClick={onCancel}>
            Cancel and go back
          </Button>
        </div>
      </CardContent>
    </Card>
  );
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
