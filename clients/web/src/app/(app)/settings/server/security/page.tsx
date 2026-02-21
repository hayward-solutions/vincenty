"use client";

import { useServerSettings } from "@/lib/hooks/use-mfa";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";

export default function ServerSecuritySettingsPage() {
  const { settings, isLoading, update } = useServerSettings();

  async function handleToggleMFA() {
    if (!settings) return;
    const newValue = !settings.mfa_required;

    if (
      newValue &&
      !confirm(
        "Enable mandatory MFA for all users? Users without MFA will be required to set it up before accessing the system."
      )
    ) {
      return;
    }

    try {
      await update({ mfa_required: newValue });
      toast.success(
        newValue
          ? "MFA is now required for all users"
          : "MFA requirement removed"
      );
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to update settings"
      );
    }
  }

  return (
    <div className="p-4 md:p-6 space-y-6 max-w-2xl">
      <h1 className="text-2xl font-semibold">Server Security</h1>

      {isLoading ? (
        <Skeleton className="h-40 w-full" />
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>Multi-Factor Authentication Policy</CardTitle>
            <CardDescription>
              Require all users to configure MFA before accessing the system.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <Label className="text-sm font-medium">
                  Require MFA for all users
                </Label>
                <p className="text-xs text-muted-foreground mt-1">
                  When enabled, users without MFA configured will be prompted to
                  set it up and cannot access other features until they do.
                </p>
              </div>
              <Button
                variant={settings?.mfa_required ? "destructive" : "default"}
                size="sm"
                onClick={handleToggleMFA}
              >
                {settings?.mfa_required ? "Disable" : "Enable"}
              </Button>
            </div>
            {settings?.mfa_required && (
              <div className="rounded-md bg-amber-500/10 p-3 text-sm text-amber-700 dark:text-amber-400">
                MFA is currently required for all users. Users without MFA will
                be blocked from accessing the system until they configure it.
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
