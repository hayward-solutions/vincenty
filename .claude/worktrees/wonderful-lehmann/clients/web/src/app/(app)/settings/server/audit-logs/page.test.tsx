import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import AuditLogsSettingsPage from "./page";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockToast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({ toast: mockToast }));

const mockAuditHook = vi.hoisted(() => ({
  data: [] as unknown[],
  total: 0,
  isLoading: false,
  error: null as string | null,
  fetch: vi.fn(),
}));

const mockExportFn = vi.hoisted(() => vi.fn());

vi.mock("@/lib/hooks/use-audit-logs", () => ({
  useAllAuditLogs: () => mockAuditHook,
  exportAllAuditLogs: mockExportFn,
}));

vi.mock("@/components/audit/audit-log-table", () => ({
  AuditLogTable: ({ logs, showUser }: { logs: unknown[]; showUser?: boolean }) => (
    <div data-testid="audit-log-table">
      Logs: {logs.length} {showUser && "(with users)"}
    </div>
  ),
}));

vi.mock("@/components/audit/audit-filters", () => ({
  AuditFilterBar: ({
    onApply,
  }: {
    onApply: (f: Record<string, string>) => void;
  }) => (
    <button data-testid="filter-bar" onClick={() => onApply({ action: "login" })}>
      Apply Filter
    </button>
  ),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
  usePathname: () => "/settings/server/audit-logs",
}));

vi.mock("next/link", () => ({
  default: ({ children, href }: { children: React.ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  ),
}));

beforeEach(() => {
  vi.clearAllMocks();
  mockAuditHook.data = [];
  mockAuditHook.total = 0;
  mockAuditHook.isLoading = false;
  mockAuditHook.error = null;
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("AuditLogsSettingsPage", () => {
  it("renders the Audit Logs heading", () => {
    render(<AuditLogsSettingsPage />);
    expect(screen.getByText("Audit Logs")).toBeInTheDocument();
  });

  it("renders export buttons", () => {
    render(<AuditLogsSettingsPage />);
    expect(screen.getByRole("button", { name: /export csv/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /export json/i })).toBeInTheDocument();
  });

  it("passes showUser to AuditLogTable", () => {
    mockAuditHook.data = [{ id: "1" }];
    render(<AuditLogsSettingsPage />);
    expect(screen.getByText(/\(with users\)/)).toBeInTheDocument();
  });

  it("calls fetch on mount", () => {
    render(<AuditLogsSettingsPage />);
    expect(mockAuditHook.fetch).toHaveBeenCalledWith(
      expect.objectContaining({ page: 1, page_size: 20 })
    );
  });

  it("shows error when present", () => {
    mockAuditHook.error = "Fetch failed";
    render(<AuditLogsSettingsPage />);
    expect(screen.getByText("Fetch failed")).toBeInTheDocument();
  });

  it("exports CSV successfully", async () => {
    mockExportFn.mockResolvedValue(undefined);
    const user = userEvent.setup();
    render(<AuditLogsSettingsPage />);
    await user.click(screen.getByRole("button", { name: /export csv/i }));
    await waitFor(() => {
      expect(mockExportFn).toHaveBeenCalledWith("csv", expect.any(Object));
      expect(mockToast.success).toHaveBeenCalledWith("Exported as CSV");
    });
  });

  it("shows error toast on export failure", async () => {
    mockExportFn.mockRejectedValue(new Error("fail"));
    const user = userEvent.setup();
    render(<AuditLogsSettingsPage />);
    await user.click(screen.getByRole("button", { name: /export csv/i }));
    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith("Export failed");
    });
  });

  it("shows pagination when total exceeds page size", () => {
    mockAuditHook.total = 25;
    mockAuditHook.data = Array(20).fill({ id: "x" });
    render(<AuditLogsSettingsPage />);
    expect(screen.getByText(/showing 1-20 of 25/i)).toBeInTheDocument();
  });
});
