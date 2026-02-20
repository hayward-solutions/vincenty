"use client";

import { useAuth } from "@/lib/auth-context";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

export default function DashboardPage() {
  const { user } = useAuth();

  return (
    <div className="p-6 space-y-6">
      <h1 className="text-2xl font-semibold">Dashboard</h1>
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Signed in as
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-lg font-semibold">
              {user?.display_name || user?.username}
            </div>
            <div className="flex items-center gap-2 mt-1">
              <span className="text-sm text-muted-foreground">{user?.email}</span>
              {user?.is_admin && <Badge variant="secondary">Admin</Badge>}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Map
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              Situational awareness map view coming in Phase 5.
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Messages
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              Secure messaging coming in Phase 6.
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
