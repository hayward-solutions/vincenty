"use client";

import { useState } from "react";
import { useTOTPSetup } from "@/lib/hooks/use-mfa";
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
import { QRCodeSVG } from "qrcode.react";
import { RecoveryCodesDisplay } from "./recovery-codes-dialog";
import type { TOTPSetupResponse } from "@/types/api";

interface TOTPSetupDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onComplete: () => void;
}

type Step = "name" | "scan" | "verify" | "recovery";

export function TOTPSetupDialog({
  open,
  onOpenChange,
  onComplete,
}: TOTPSetupDialogProps) {
  const { beginSetup, verifySetup, isLoading } = useTOTPSetup();
  const [step, setStep] = useState<Step>("name");
  const [name, setName] = useState("Authenticator App");
  const [setup, setSetup] = useState<TOTPSetupResponse | null>(null);
  const [code, setCode] = useState("");
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);

  function reset() {
    setStep("name");
    setName("Authenticator App");
    setSetup(null);
    setCode("");
    setRecoveryCodes([]);
  }

  function handleClose(open: boolean) {
    if (!open) reset();
    onOpenChange(open);
  }

  async function handleBeginSetup(e: React.FormEvent) {
    e.preventDefault();
    try {
      const resp = await beginSetup(name);
      setSetup(resp);
      setStep("scan");
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to begin TOTP setup"
      );
    }
  }

  async function handleVerify(e: React.FormEvent) {
    e.preventDefault();
    if (!setup) return;
    try {
      const resp = await verifySetup(setup.method_id, code);
      if (resp.recovery_codes && resp.recovery_codes.length > 0) {
        setRecoveryCodes(resp.recovery_codes);
        setStep("recovery");
      } else {
        toast.success("Authenticator app configured");
        handleClose(false);
        onComplete();
      }
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Invalid code"
      );
    }
  }

  function handleRecoveryDone() {
    toast.success("Authenticator app configured");
    handleClose(false);
    onComplete();
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>
            {step === "name" && "Set Up Authenticator App"}
            {step === "scan" && "Scan QR Code"}
            {step === "verify" && "Verify Code"}
            {step === "recovery" && "Save Recovery Codes"}
          </DialogTitle>
        </DialogHeader>

        {step === "name" && (
          <form onSubmit={handleBeginSetup} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="totp-name">Device Name</Label>
              <Input
                id="totp-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. Google Authenticator"
                required
              />
              <p className="text-xs text-muted-foreground">
                A name to identify this authenticator.
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
                Continue
              </Button>
            </DialogFooter>
          </form>
        )}

        {step === "scan" && setup && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Scan this QR code with your authenticator app, or enter the secret
              key manually.
            </p>
            <div className="flex justify-center">
              <QRCodeSVG value={setup.uri} size={200} />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Secret Key</Label>
              <code className="block bg-muted p-2 rounded text-xs font-mono break-all select-all">
                {setup.secret}
              </code>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setStep("name")}>
                Back
              </Button>
              <Button onClick={() => setStep("verify")}>
                I&apos;ve scanned it
              </Button>
            </DialogFooter>
          </div>
        )}

        {step === "verify" && (
          <form onSubmit={handleVerify} className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Enter the 6-digit code from your authenticator app to complete
              setup.
            </p>
            <div className="space-y-2">
              <Label htmlFor="totp-code">Verification Code</Label>
              <Input
                id="totp-code"
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                placeholder="000000"
                maxLength={6}
                className="text-center text-lg font-mono tracking-widest"
                autoFocus
                required
              />
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setStep("scan")}
              >
                Back
              </Button>
              <Button type="submit" disabled={isLoading || code.length !== 6}>
                {isLoading ? "Verifying..." : "Verify"}
              </Button>
            </DialogFooter>
          </form>
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
