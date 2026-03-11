"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

interface ApiInfo {
  service: string;
  version: string;
}

interface VersionRowProps {
  label: string;
  value: string | null;
  loading?: boolean;
}

function VersionRow({ label, value, loading }: VersionRowProps) {
  return (
    <div className="flex items-center justify-between py-3 border-b last:border-0">
      <span className="text-sm font-medium">{label}</span>
      {loading ? (
        <Skeleton className="h-4 w-24" />
      ) : (
        <span className="text-sm text-muted-foreground font-mono">
          {value ?? "—"}
        </span>
      )}
    </div>
  );
}

export default function AboutSettingsPage() {
  const [apiInfo, setApiInfo] = useState<ApiInfo | null>(null);
  const [apiLoading, setApiLoading] = useState(true);
  const [apiError, setApiError] = useState(false);

  const webVersion = process.env.NEXT_PUBLIC_APP_VERSION ?? "dev";

  useEffect(() => {
    let cancelled = false;
    setApiLoading(true);
    setApiError(false);

    api
      .get<ApiInfo>("/api/v1")
      .then((data) => {
        if (!cancelled) setApiInfo(data);
      })
      .catch(() => {
        if (!cancelled) setApiError(true);
      })
      .finally(() => {
        if (!cancelled) setApiLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="p-4 md:p-6 space-y-6">
      <h1 className="text-2xl font-semibold">About</h1>

      <Card>
        <CardHeader>
          <CardTitle>Version Information</CardTitle>
          <CardDescription>
            Software versions for each component of this Vincenty instance.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <VersionRow label="Web Client" value={webVersion} />
          <VersionRow
            label="API Server"
            value={
              apiError
                ? "unavailable"
                : apiInfo?.version ?? null
            }
            loading={apiLoading}
          />
        </CardContent>
      </Card>
    </div>
  );
}
