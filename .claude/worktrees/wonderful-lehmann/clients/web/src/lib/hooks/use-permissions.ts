"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { PermissionPolicy } from "@/types/api";

export function usePermissionPolicy() {
  const [policy, setPolicy] = useState<PermissionPolicy | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const fetch = useCallback(async () => {
    setIsLoading(true);
    try {
      const data = await api.get<PermissionPolicy>(
        "/api/v1/server/permissions"
      );
      setPolicy(data);
    } catch {
      // May not be admin
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetch();
  }, [fetch]);

  const update = useCallback(
    async (updates: PermissionPolicy) => {
      setIsLoading(true);
      try {
        const data = await api.put<PermissionPolicy>(
          "/api/v1/server/permissions",
          updates
        );
        setPolicy(data);
        return data;
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  return { policy, isLoading, update, refetch: fetch };
}
