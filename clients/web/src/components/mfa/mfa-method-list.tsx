"use client";

import { useState } from "react";
import {
  useMFAMethods,
  useDeleteMFAMethod,
  useTogglePasswordless,
} from "@/lib/hooks/use-mfa";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";
import { TOTPSetupDialog } from "./totp-setup-dialog";
import { WebAuthnRegisterDialog } from "./webauthn-register-dialog";
import { RegenerateRecoveryCodesDialog } from "./recovery-codes-dialog";
import type { MFAMethod } from "@/types/api";

export function MFAMethodList() {
  const { methods, isLoading, refetch } = useMFAMethods();
  const [totpSetupOpen, setTotpSetupOpen] = useState(false);
  const [webauthnSetupOpen, setWebauthnSetupOpen] = useState(false);
  const [recoveryCodesOpen, setRecoveryCodesOpen] = useState(false);

  const hasMethods = methods.length > 0;
  const totpMethods = methods.filter((m) => m.type === "totp");
  const webauthnMethods = methods.filter((m) => m.type === "webauthn");

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Multi-Factor Authentication</CardTitle>
          <CardDescription>
            Add an extra layer of security to your account.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {isLoading ? (
            <div className="space-y-2">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : (
            <>
              {/* TOTP section */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-sm font-medium">Authenticator App</h3>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setTotpSetupOpen(true)}
                  >
                    Add
                  </Button>
                </div>
                {totpMethods.length > 0 ? (
                  <div className="space-y-2">
                    {totpMethods.map((m) => (
                      <MethodRow key={m.id} method={m} onDeleted={refetch} />
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">
                    No authenticator apps configured.
                  </p>
                )}
              </div>

              {/* WebAuthn section */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-sm font-medium">Security Keys & Passkeys</h3>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setWebauthnSetupOpen(true)}
                  >
                    Add
                  </Button>
                </div>
                {webauthnMethods.length > 0 ? (
                  <div className="space-y-2">
                    {webauthnMethods.map((m) => (
                      <MethodRow key={m.id} method={m} onDeleted={refetch} />
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">
                    No security keys or passkeys registered.
                  </p>
                )}
              </div>

              {/* Recovery codes */}
              {hasMethods && (
                <div className="pt-2 border-t">
                  <div className="flex items-center justify-between">
                    <div>
                      <h3 className="text-sm font-medium">Recovery Codes</h3>
                      <p className="text-xs text-muted-foreground">
                        One-time codes for account recovery.
                      </p>
                    </div>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setRecoveryCodesOpen(true)}
                    >
                      Regenerate
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>

      <TOTPSetupDialog
        open={totpSetupOpen}
        onOpenChange={setTotpSetupOpen}
        onComplete={refetch}
      />
      <WebAuthnRegisterDialog
        open={webauthnSetupOpen}
        onOpenChange={setWebauthnSetupOpen}
        onComplete={refetch}
      />
      <RegenerateRecoveryCodesDialog
        open={recoveryCodesOpen}
        onOpenChange={setRecoveryCodesOpen}
      />
    </>
  );
}

function MethodRow({
  method,
  onDeleted,
}: {
  method: MFAMethod;
  onDeleted: () => void;
}) {
  const { deleteMethod, isLoading: deleteLoading } = useDeleteMFAMethod();
  const { toggle, isLoading: toggleLoading } = useTogglePasswordless();

  async function handleDelete() {
    if (!confirm(`Remove "${method.name}"? You may lose access to your account.`))
      return;
    try {
      await deleteMethod(method.id);
      toast.success(`"${method.name}" removed`);
      onDeleted();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to remove method"
      );
    }
  }

  async function handleTogglePasswordless() {
    try {
      await toggle(method.id, !method.passwordless_enabled);
      toast.success(
        method.passwordless_enabled
          ? "Passwordless login disabled"
          : "Passwordless login enabled"
      );
      onDeleted();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to update"
      );
    }
  }

  return (
    <div className="flex items-center justify-between rounded-md border p-3">
      <div className="flex items-center gap-3">
        <span className="text-lg">
          {method.type === "totp" ? "📱" : "🔑"}
        </span>
        <div>
          <p className="text-sm font-medium">{method.name}</p>
          <p className="text-xs text-muted-foreground">
            Added{" "}
            {new Date(method.created_at).toLocaleDateString()}
            {method.last_used_at &&
              ` · Last used ${new Date(method.last_used_at).toLocaleDateString()}`}
          </p>
        </div>
      </div>
      <div className="flex items-center gap-2">
        {method.type === "webauthn" && (
          <Badge
            variant={method.passwordless_enabled ? "default" : "outline"}
            className="cursor-pointer select-none"
            onClick={handleTogglePasswordless}
          >
            {toggleLoading
              ? "..."
              : method.passwordless_enabled
                ? "Passwordless"
                : "MFA only"}
          </Badge>
        )}
        <Button
          variant="ghost"
          size="sm"
          className="text-destructive"
          onClick={handleDelete}
          disabled={deleteLoading}
        >
          Remove
        </Button>
      </div>
    </div>
  );
}
