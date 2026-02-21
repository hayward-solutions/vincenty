"use client";

import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import {
  useAllAuditLogs,
  exportAllAuditLogs,
} from "@/lib/hooks/use-audit-logs";
import { AuditLogTable } from "@/components/audit/audit-log-table";
import { AuditFilterBar } from "@/components/audit/audit-filters";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import type { AuditFilters } from "@/types/api";

export default function AuditLogsSettingsPage() {
  const { data, total, isLoading, error, fetch } = useAllAuditLogs();
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState<AuditFilters>({});
  const pageSize = 20;

  const loadPage = useCallback(
    (p: number, f: AuditFilters) => {
      fetch({ ...f, page: p, page_size: pageSize });
    },
    [fetch]
  );

  useEffect(() => {
    loadPage(page, filters);
  }, [page, filters, loadPage]);

  function handleFilter(f: AuditFilters) {
    setPage(1);
    setFilters(f);
  }

  async function handleExport(format: "csv" | "json") {
    try {
      await exportAllAuditLogs(format, filters);
      toast.success(`Exported as ${format.toUpperCase()}`);
    } catch {
      toast.error("Export failed");
    }
  }

  return (
    <div className="p-4 md:p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Audit Logs</h1>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleExport("csv")}
          >
            Export CSV
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleExport("json")}
          >
            Export JSON
          </Button>
        </div>
      </div>

      <AuditFilterBar onApply={handleFilter} />

      {error && <p className="text-sm text-destructive">{error}</p>}

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => {
            const key = `skeleton-${i}`;
            return <Skeleton key={key} className="h-12 w-full" />;
          })}
        </div>
      ) : (
        <>
          <AuditLogTable logs={data} showUser />

          {total > pageSize && (
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                Showing {(page - 1) * pageSize + 1}-
                {Math.min(page * pageSize, total)} of {total}
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage(page - 1)}
                >
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page * pageSize >= total}
                  onClick={() => setPage(page + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
