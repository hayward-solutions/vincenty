"use client";

import { useState } from "react";
import { useRecoveryCodes } from "@/lib/hooks/use-mfa";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";

// ---------------------------------------------------------------------------
// Inline display (used inside other dialogs)
// ---------------------------------------------------------------------------

export function RecoveryCodesDisplay({
  codes,
  onDone,
}: {
  codes: string[];
  onDone: () => void;
}) {
  const [confirmed, setConfirmed] = useState(false);

  function handleCopy() {
    navigator.clipboard.writeText(codes.join("\n"));
    toast.success("Recovery codes copied to clipboard");
  }

  function handleDownload() {
    const blob = new Blob([codes.join("\n")], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "vincenty-recovery-codes.txt";
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Save these recovery codes in a safe place. Each code can only be used
        once to sign in if you lose access to your authenticator.
      </p>
      <div className="grid grid-cols-2 gap-2 p-3 bg-muted rounded-md">
        {codes.map((code, i) => (
          <code key={i} className="text-sm font-mono text-center py-1">
            {code}
          </code>
        ))}
      </div>
      <div className="flex gap-2">
        <Button variant="outline" size="sm" onClick={handleCopy}>
          Copy All
        </Button>
        <Button variant="outline" size="sm" onClick={handleDownload}>
          Download
        </Button>
      </div>
      <div className="flex items-center gap-2">
        <input
          type="checkbox"
          id="confirm-saved"
          checked={confirmed}
          onChange={(e) => setConfirmed(e.target.checked)}
          className="h-4 w-4"
        />
        <label htmlFor="confirm-saved" className="text-sm">
          I have saved these recovery codes
        </label>
      </div>
      <DialogFooter>
        <Button onClick={onDone} disabled={!confirmed}>
          Done
        </Button>
      </DialogFooter>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Standalone regeneration dialog
// ---------------------------------------------------------------------------

interface RegenerateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function RegenerateRecoveryCodesDialog({
  open,
  onOpenChange,
}: RegenerateDialogProps) {
  const { regenerate, isLoading } = useRecoveryCodes();
  const [codes, setCodes] = useState<string[] | null>(null);

  async function handleRegenerate() {
    try {
      const resp = await regenerate();
      setCodes(resp.codes);
    } catch (err) {
      toast.error(
        err instanceof ApiError
          ? err.message
          : "Failed to regenerate recovery codes"
      );
    }
  }

  function handleClose(open: boolean) {
    if (!open) setCodes(null);
    onOpenChange(open);
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Recovery Codes</DialogTitle>
        </DialogHeader>

        {codes ? (
          <RecoveryCodesDisplay
            codes={codes}
            onDone={() => handleClose(false)}
          />
        ) : (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Regenerating recovery codes will invalidate all existing codes.
              Make sure you have access to your authenticator before proceeding.
            </p>
            <DialogFooter>
              <Button variant="outline" onClick={() => handleClose(false)}>
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={handleRegenerate}
                disabled={isLoading}
              >
                {isLoading ? "Regenerating..." : "Regenerate Codes"}
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
