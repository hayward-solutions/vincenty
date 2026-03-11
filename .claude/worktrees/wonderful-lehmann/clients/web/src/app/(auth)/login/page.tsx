"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { MFAChallenge } from "@/components/mfa/mfa-challenge";
import type { AuthResponse, MFAChallengeResponse } from "@/types/api";
import { isMFAChallengeResponse } from "@/types/api";

export default function LoginPage() {
  const router = useRouter();
  const { login, completeMFALogin, passkeyLogin } = useAuth();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [mfaChallenge, setMfaChallenge] = useState<MFAChallengeResponse | null>(
    null
  );

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setIsLoading(true);

    try {
      const result = await login(username, password);

      // Check if MFA is required
      if (result && isMFAChallengeResponse(result)) {
        setMfaChallenge(result);
        setIsLoading(false);
        return;
      }

      router.push("/dashboard");
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("An unexpected error occurred");
      }
    } finally {
      setIsLoading(false);
    }
  }

  function handleMFASuccess(resp: AuthResponse) {
    completeMFALogin(resp);
    router.push("/dashboard");
  }

  function handleMFACancel() {
    setMfaChallenge(null);
    setPassword("");
  }

  async function handlePasskeyLogin() {
    setError("");
    setIsLoading(true);
    try {
      await passkeyLogin();
      router.push("/dashboard");
    } catch (err) {
      if (err instanceof DOMException && err.name === "NotAllowedError") {
        // User cancelled
      } else if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Passkey login failed");
      }
    } finally {
      setIsLoading(false);
    }
  }

  // Show MFA challenge screen
  if (mfaChallenge) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-background">
        <MFAChallenge
          challenge={mfaChallenge}
          onSuccess={handleMFASuccess}
          onCancel={handleMFACancel}
        />
      </main>
    );
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-background">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle className="text-center text-2xl">Vincenty</CardTitle>
          <p className="text-center text-sm text-muted-foreground">
            Sign in to continue
          </p>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
              <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                {error}
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="Enter your username"
                required
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Enter your password"
                required
              />
            </div>
            <Button type="submit" className="w-full" disabled={isLoading}>
              {isLoading ? "Signing in..." : "Sign in"}
            </Button>
          </form>

          <div className="relative my-4">
            <div className="absolute inset-0 flex items-center">
              <span className="w-full border-t" />
            </div>
            <div className="relative flex justify-center text-xs uppercase">
              <span className="bg-card px-2 text-muted-foreground">or</span>
            </div>
          </div>

          <Button
            variant="outline"
            className="w-full"
            onClick={handlePasskeyLogin}
            disabled={isLoading}
          >
            Sign in with Passkey
          </Button>
        </CardContent>
      </Card>
    </main>
  );
}
