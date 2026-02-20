"use client";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export default function DevicesSettingsPage() {
  return (
    <div className="p-6 space-y-6 max-w-2xl">
      <h1 className="text-2xl font-semibold">Devices</h1>

      <Card>
        <CardHeader>
          <CardTitle>Your Devices</CardTitle>
          <CardDescription>
            Manage the devices that are signed in to your account.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            Device management coming soon.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
