"use client";

import { useState } from "react";
import { useWebAuthnRegister } from "@/lib/hooks/use-mfa";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";
import { RecoveryCodesDisplay } from "./recovery-codes-dialog";

interface WebAuthnRegisterDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onComplete: () => void;
}

type Step = "name" | "waiting" | "recovery";

export function WebAuthnRegisterDialog({
  open,
  onOpenChange,
  onComplete,
}: WebAuthnRegisterDialogProps) {
  const { beginRegister, finishRegister, isLoading } = useWebAuthnRegister();
  const [step, setStep] = useState<Step>("name");
  const [name, setName] = useState("Security Key");
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);

  function reset() {
    setStep("name");
    setName("Security Key");
    setRecoveryCodes([]);
  }

  function handleClose(open: boolean) {
    if (!open) reset();
    onOpenChange(open);
  }

  async function handleRegister(e: React.FormEvent) {
    e.preventDefault();
    try {
      setStep("waiting");

      // Get creation options from server (wrapped in { publicKey: { ... } })
      const serverResp = await beginRegister(name);

      // Extract the publicKey options and convert base64url fields to ArrayBuffers
      // so the browser WebAuthn API can process them.
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const opts = serverResp.publicKey as any;
      const user = opts.user as Record<string, unknown>;
      const publicKeyOptions: PublicKeyCredentialCreationOptions = {
        ...opts,
        challenge: base64URLToBuffer(opts.challenge as string),
        user: {
          ...user,
          id: base64URLToBuffer(user.id as string),
        } as PublicKeyCredentialUserEntity,
        excludeCredentials: (
          (opts.excludeCredentials as Array<Record<string, unknown>>) ?? []
        ).map((c) => ({
          ...c,
          id: base64URLToBuffer(c.id as string),
        })) as PublicKeyCredentialDescriptor[],
      } as PublicKeyCredentialCreationOptions;

      // Call the WebAuthn browser API
      const credential = await navigator.credentials.create({
        publicKey: publicKeyOptions,
      });

      if (!credential) {
        throw new Error("No credential returned from browser");
      }

      // Send the attestation to the server
      const pkCred = credential as PublicKeyCredential;
      const attestation =
        pkCred.response as AuthenticatorAttestationResponse;

      const resp = await finishRegister({
        id: pkCred.id,
        rawId: bufferToBase64URL(pkCred.rawId),
        type: pkCred.type,
        response: {
          attestationObject: bufferToBase64URL(
            attestation.attestationObject
          ),
          clientDataJSON: bufferToBase64URL(attestation.clientDataJSON),
        },
      } as unknown as Credential);

      if (resp.recovery_codes && resp.recovery_codes.length > 0) {
        setRecoveryCodes(resp.recovery_codes);
        setStep("recovery");
      } else {
        toast.success("Security key registered");
        handleClose(false);
        onComplete();
      }
    } catch (err) {
      setStep("name");
      if (err instanceof ApiError) {
        toast.error(err.message);
      } else if (err instanceof DOMException && err.name === "NotAllowedError") {
        toast.error("Registration was cancelled");
      } else {
        toast.error("Failed to register security key");
      }
    }
  }

  function handleRecoveryDone() {
    toast.success("Security key registered");
    handleClose(false);
    onComplete();
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>
            {step === "name" && "Register Security Key"}
            {step === "waiting" && "Touch Your Security Key"}
            {step === "recovery" && "Save Recovery Codes"}
          </DialogTitle>
        </DialogHeader>

        {step === "name" && (
          <form onSubmit={handleRegister} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="webauthn-name">Credential Name</Label>
              <Input
                id="webauthn-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. YubiKey 5"
                required
              />
              <p className="text-xs text-muted-foreground">
                A name to identify this security key or passkey.
              </p>
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => handleClose(false)}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={isLoading}>
                Register
              </Button>
            </DialogFooter>
          </form>
        )}

        {step === "waiting" && (
          <div className="space-y-4 text-center py-8">
            <div className="text-4xl">🔑</div>
            <p className="text-sm text-muted-foreground">
              Follow your browser&apos;s prompt to complete registration.
            </p>
          </div>
        )}

        {step === "recovery" && (
          <RecoveryCodesDisplay
            codes={recoveryCodes}
            onDone={handleRecoveryDone}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}

/** Convert a base64url-encoded string to an ArrayBuffer. */
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

/** Convert an ArrayBuffer to a base64url-encoded string. */
function bufferToBase64URL(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let str = "";
  for (const b of bytes) {
    str += String.fromCharCode(b);
  }
  return btoa(str).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}
