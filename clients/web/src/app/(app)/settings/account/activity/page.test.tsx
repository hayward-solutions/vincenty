import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import ActivitySettingsPage from "./page";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockToast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({
  toast: mockToast,
}));

const mockAuditHook = vi.hoisted(() => ({
  data: [] as unknown[],
  total: 0,
  isLoading: false,
  error: null as string | null,
  fetch: vi.fn(),
}));

const mockExportFn = vi.hoisted(() => vi.fn());

vi.mock("@/lib/hooks/use-audit-logs", () => ({
  useMyAuditLogs: () => mockAuditHook,
  exportMyAuditLogs: mockExportFn,
}));

// Mock child components
vi.mock("@/components/audit/audit-log-table", () => ({
  AuditLogTable: ({ logs }: { logs: unknown[] }) => (
    <div data-testid="audit-log-table">Logs: {logs.length}</div>
  ),
}));

vi.mock("@/components/audit/audit-filters", () => ({
  AuditFilterBar: ({
    onApply,
  }: {
    onApply: (f: Record<string, string>) => void;
  }) => (
    <button
      data-testid="filter-bar"
      onClick={() => onApply({ action: "login" })}
    >
      Apply Filter
    </button>
  ),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
  usePathname: () => "/settings/account/activity",
}));

vi.mock("next/link", () => ({
  default: ({
    children,
    href,
  }: {
    children: React.ReactNode;
    href: string;
  }) => <a href={href}>{children}</a>,
}));

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

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

describe("ActivitySettingsPage", () => {
  it("renders the Activity heading", () => {
    render(<ActivitySettingsPage />);
    expect(screen.getByText("Activity")).toBeInTheDocument();
  });

  it("renders Export CSV and Export JSON buttons", () => {
    render(<ActivitySettingsPage />);
    expect(
      screen.getByRole("button", { name: /export csv/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /export json/i })
    ).toBeInTheDocument();
  });

  it("renders the filter bar", () => {
    render(<ActivitySettingsPage />);
    expect(screen.getByTestId("filter-bar")).toBeInTheDocument();
  });

  it("calls fetch on mount", () => {
    render(<ActivitySettingsPage />);
    expect(mockAuditHook.fetch).toHaveBeenCalledWith(
      expect.objectContaining({ page: 1, page_size: 20 })
    );
  });

  it("renders audit log table with data", () => {
    mockAuditHook.data = [{ id: "1" }, { id: "2" }];
    render(<ActivitySettingsPage />);
    expect(screen.getByTestId("audit-log-table")).toBeInTheDocument();
    expect(screen.getByText("Logs: 2")).toBeInTheDocument();
  });

  it("shows error message when error exists", () => {
    mockAuditHook.error = "Something went wrong";
    render(<ActivitySettingsPage />);
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Pagination
  // -----------------------------------------------------------------------

  it("shows pagination when total > pageSize", () => {
    mockAuditHook.total = 25;
    mockAuditHook.data = Array(20).fill({ id: "x" });
    render(<ActivitySettingsPage />);
    expect(screen.getByText(/showing 1-20 of 25/i)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /next/i })
    ).toBeInTheDocument();
  });

  it("Previous button is disabled on first page", () => {
    mockAuditHook.total = 25;
    mockAuditHook.data = Array(20).fill({ id: "x" });
    render(<ActivitySettingsPage />);
    expect(
      screen.getByRole("button", { name: /previous/i })
    ).toBeDisabled();
  });

  it("calls fetch with next page on Next click", async () => {
    mockAuditHook.total = 25;
    mockAuditHook.data = Array(20).fill({ id: "x" });
    const user = userEvent.setup();
    render(<ActivitySettingsPage />);

    await user.click(screen.getByRole("button", { name: /next/i }));

    await waitFor(() => {
      expect(mockAuditHook.fetch).toHaveBeenCalledWith(
        expect.objectContaining({ page: 2 })
      );
    });
  });

  // -----------------------------------------------------------------------
  // Filters
  // -----------------------------------------------------------------------

  it("resets page and applies filters", async () => {
    mockAuditHook.total = 25;
    mockAuditHook.data = Array(20).fill({ id: "x" });
    const user = userEvent.setup();
    render(<ActivitySettingsPage />);

    await user.click(screen.getByTestId("filter-bar"));

    await waitFor(() => {
      expect(mockAuditHook.fetch).toHaveBeenCalledWith(
        expect.objectContaining({ action: "login", page: 1, page_size: 20 })
      );
    });
  });

  // -----------------------------------------------------------------------
  // Export
  // -----------------------------------------------------------------------

  it("calls exportMyAuditLogs with csv and shows success toast", async () => {
    mockExportFn.mockResolvedValue(undefined);
    const user = userEvent.setup();
    render(<ActivitySettingsPage />);

    await user.click(
      screen.getByRole("button", { name: /export csv/i })
    );

    await waitFor(() => {
      expect(mockExportFn).toHaveBeenCalledWith("csv", expect.any(Object));
      expect(mockToast.success).toHaveBeenCalledWith("Exported as CSV");
    });
  });

  it("calls exportMyAuditLogs with json and shows success toast", async () => {
    mockExportFn.mockResolvedValue(undefined);
    const user = userEvent.setup();
    render(<ActivitySettingsPage />);

    await user.click(
      screen.getByRole("button", { name: /export json/i })
    );

    await waitFor(() => {
      expect(mockExportFn).toHaveBeenCalledWith("json", expect.any(Object));
      expect(mockToast.success).toHaveBeenCalledWith("Exported as JSON");
    });
  });

  it("shows error toast when export fails", async () => {
    mockExportFn.mockRejectedValue(new Error("fail"));
    const user = userEvent.setup();
    render(<ActivitySettingsPage />);

    await user.click(
      screen.getByRole("button", { name: /export csv/i })
    );

    await waitFor(() => {
      expect(mockToast.error).toHaveBeenCalledWith("Export failed");
    });
  });
});
