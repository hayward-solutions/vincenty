"use client";

import { useCallback, useState } from "react";
import { api } from "@/lib/api";
import type {
  AuditLogResponse,
  AuditFilters,
  ListResponse,
} from "@/types/api";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

// ---------------------------------------------------------------------------
// My audit logs (authenticated user)
// ---------------------------------------------------------------------------

export function useMyAuditLogs() {
  const [data, setData] = useState<AuditLogResponse[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetch = useCallback(async (filters?: AuditFilters) => {
    setIsLoading(true);
    setError(null);
    try {
      const params: Record<string, string> = {};
      if (filters?.from) params.from = filters.from;
      if (filters?.to) params.to = filters.to;
      if (filters?.action) params.action = filters.action;
      if (filters?.resource_type) params.resource_type = filters.resource_type;
      if (filters?.page) params.page = String(filters.page);
      if (filters?.page_size) params.page_size = String(filters.page_size);

      const result = await api.get<ListResponse<AuditLogResponse>>(
        "/api/v1/audit-logs/me",
        { params }
      );
      setData(result.data ?? []);
      setTotal(result.total);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch audit logs"
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  return { data, total, isLoading, error, fetch };
}

// ---------------------------------------------------------------------------
// Group audit logs (group admin or system admin)
// ---------------------------------------------------------------------------

export function useGroupAuditLogs() {
  const [data, setData] = useState<AuditLogResponse[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetch = useCallback(
    async (groupId: string, filters?: AuditFilters) => {
      setIsLoading(true);
      setError(null);
      try {
        const params: Record<string, string> = {};
        if (filters?.from) params.from = filters.from;
        if (filters?.to) params.to = filters.to;
        if (filters?.action) params.action = filters.action;
        if (filters?.resource_type)
          params.resource_type = filters.resource_type;
        if (filters?.page) params.page = String(filters.page);
        if (filters?.page_size) params.page_size = String(filters.page_size);

        const result = await api.get<ListResponse<AuditLogResponse>>(
          `/api/v1/groups/${groupId}/audit-logs`,
          { params }
        );
        setData(result.data ?? []);
        setTotal(result.total);
      } catch (err) {
        setError(
          err instanceof Error ? err.message : "Failed to fetch group audit logs"
        );
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  return { data, total, isLoading, error, fetch };
}

// ---------------------------------------------------------------------------
// All audit logs (admin only)
// ---------------------------------------------------------------------------

export function useAllAuditLogs() {
  const [data, setData] = useState<AuditLogResponse[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetch = useCallback(async (filters?: AuditFilters) => {
    setIsLoading(true);
    setError(null);
    try {
      const params: Record<string, string> = {};
      if (filters?.from) params.from = filters.from;
      if (filters?.to) params.to = filters.to;
      if (filters?.action) params.action = filters.action;
      if (filters?.resource_type) params.resource_type = filters.resource_type;
      if (filters?.page) params.page = String(filters.page);
      if (filters?.page_size) params.page_size = String(filters.page_size);

      const result = await api.get<ListResponse<AuditLogResponse>>(
        "/api/v1/audit-logs",
        { params }
      );
      setData(result.data ?? []);
      setTotal(result.total);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch all audit logs"
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  return { data, total, isLoading, error, fetch };
}

// ---------------------------------------------------------------------------
// Export helpers
// ---------------------------------------------------------------------------

function buildFilterParams(filters?: AuditFilters): string {
  const params = new URLSearchParams();
  if (filters?.from) params.set("from", filters.from);
  if (filters?.to) params.set("to", filters.to);
  if (filters?.action) params.set("action", filters.action);
  if (filters?.resource_type)
    params.set("resource_type", filters.resource_type);
  return params.toString();
}

/**
 * Download audit logs as CSV or JSON for the current user.
 */
export async function exportMyAuditLogs(
  format: "csv" | "json",
  filters?: AuditFilters
) {
  const q = buildFilterParams(filters);
  const url = `${API_BASE}/api/v1/audit-logs/me/export?format=${format}${q ? `&${q}` : ""}`;
  const token = localStorage.getItem("access_token");
  const res = await window.fetch(url, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) throw new Error("Export failed");
  const blob = await res.blob();
  downloadBlob(
    blob,
    `my_audit_logs.${format === "csv" ? "csv" : "json"}`
  );
}

/**
 * Download audit logs as CSV or JSON for all users (admin).
 */
export async function exportAllAuditLogs(
  format: "csv" | "json",
  filters?: AuditFilters
) {
  const q = buildFilterParams(filters);
  const url = `${API_BASE}/api/v1/audit-logs/export?format=${format}${q ? `&${q}` : ""}`;
  const token = localStorage.getItem("access_token");
  const res = await window.fetch(url, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) throw new Error("Export failed");
  const blob = await res.blob();
  downloadBlob(blob, `audit_logs.${format === "csv" ? "csv" : "json"}`);
}

/**
 * Download own location history as GPX.
 */
export async function exportMyLocationGPX(from: Date, to: Date) {
  const params = new URLSearchParams({
    from: from.toISOString(),
    to: to.toISOString(),
  });
  const url = `${API_BASE}/api/v1/users/me/locations/export?${params}`;
  const token = localStorage.getItem("access_token");
  const res = await window.fetch(url, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) throw new Error("Export failed");
  const blob = await res.blob();
  downloadBlob(blob, `track.gpx`);
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}
