"use client";

import { useState } from "react";
import { useChangePassword } from "@/lib/hooks/use-profile";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";
import { MFAMethodList } from "@/components/mfa/mfa-method-list";

export default function SecuritySettingsPage() {
  const { changePassword, isLoading } = useChangePassword();

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [validationError, setValidationError] = useState<string | null>(null);

  function resetForm() {
    setCurrentPassword("");
    setNewPassword("");
    setConfirmPassword("");
    setValidationError(null);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setValidationError(null);

    // Client-side validation
    if (!currentPassword) {
      setValidationError("Current password is required");
      return;
    }
    if (newPassword.length < 8) {
      setValidationError("New password must be at least 8 characters");
      return;
    }
    if (newPassword !== confirmPassword) {
      setValidationError("New passwords do not match");
      return;
    }

    try {
      await changePassword({
        current_password: currentPassword,
        new_password: newPassword,
      });
      resetForm();
      toast.success("Password changed successfully");
    } catch (err) {
      if (err instanceof ApiError) {
        toast.error(err.message);
      } else {
        toast.error("Failed to change password");
      }
    }
  }

  return (
    <div className="p-4 md:p-6 space-y-6 max-w-2xl">
      <h1 className="text-2xl font-semibold">Security</h1>

      <Card>
        <CardHeader>
          <CardTitle>Password</CardTitle>
          <CardDescription>
            Change your password to keep your account secure.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="currentPassword">Current Password</Label>
              <Input
                id="currentPassword"
                type="password"
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
                placeholder="Enter your current password"
                autoComplete="current-password"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="newPassword">New Password</Label>
              <Input
                id="newPassword"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder="Enter your new password"
                autoComplete="new-password"
              />
              <p className="text-xs text-muted-foreground">
                Must be at least 8 characters.
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="confirmPassword">Confirm New Password</Label>
              <Input
                id="confirmPassword"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="Re-enter your new password"
                autoComplete="new-password"
              />
            </div>

            {validationError && (
              <p className="text-sm text-destructive">{validationError}</p>
            )}

            <Button type="submit" disabled={isLoading}>
              {isLoading ? "Changing..." : "Change Password"}
            </Button>
          </form>
        </CardContent>
      </Card>

      <MFAMethodList />

      <Card>
        <CardHeader>
          <CardTitle>API Tokens</CardTitle>
          <CardDescription>
            Manage tokens for API access.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            API token management coming soon.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
