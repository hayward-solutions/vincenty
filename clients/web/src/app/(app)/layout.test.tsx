import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render, mockAuth, mockWebSocket } from "@/test/test-utils";
import AppLayout from "./layout";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mocks = vi.hoisted(() => ({
  routerPush: vi.fn(),
  pathname: "/dashboard",
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mocks.routerPush }),
  usePathname: () => mocks.pathname,
}));

vi.mock("next/link", () => ({
  default: ({
    children,
    href,
    ...props
  }: {
    children: React.ReactNode;
    href: string;
    [key: string]: any;
  }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

vi.mock("@/lib/hooks/use-location-sharing", () => ({
  useLocationSharing: () => ({ error: mocks.locationError }),
}));

// Extend the hoisted mocks object to include locationError
mocks.locationError = null as string | null;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Save original mock values so we can restore after each test. */
const originalAuth = { ...mockAuth, user: { ...mockAuth.user } };
const originalWS = { ...mockWebSocket };

afterEach(() => {
  // Restore auth
  mockAuth.isLoading = originalAuth.isLoading;
  mockAuth.isAuthenticated = originalAuth.isAuthenticated;
  mockAuth.isAdmin = originalAuth.isAdmin;
  mockAuth.user = { ...originalAuth.user };
  mockAuth.logout = vi.fn();

  // Restore websocket
  mockWebSocket.connectionState = originalWS.connectionState;

  // Restore navigation mocks
  mocks.routerPush.mockReset();
  mocks.pathname = "/dashboard";
  mocks.locationError = null;
});

function renderLayout(children: React.ReactNode = <p>page content</p>) {
  return render(<AppLayout>{children}</AppLayout>);
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("AppLayout", () => {
  // 1. Brand link
  it('renders "SitAware" brand link pointing to /dashboard', () => {
    renderLayout();
    const brand = screen.getByRole("link", { name: "SitAware" });
    expect(brand).toBeInTheDocument();
    expect(brand).toHaveAttribute("href", "/dashboard");
  });

  // 2. Nav items
  it("renders nav items: Dashboard, Map, Messages", () => {
    renderLayout();
    expect(screen.getAllByText("Dashboard").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Map").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Messages").length).toBeGreaterThanOrEqual(1);
  });

  // 3. Children rendered
  it("renders children in the main section", () => {
    renderLayout(<p>page content</p>);
    expect(screen.getByText("page content")).toBeInTheDocument();
    // Verify it's inside a <main> element
    const main = screen.getByRole("main");
    expect(main).toContainElement(screen.getByText("page content"));
  });

  // 4. User initials
  it('shows user initials "TU" in avatar fallback', () => {
    renderLayout();
    // Initials appear in both desktop avatar and mobile sheet avatar
    const initialsElements = screen.getAllByText("TU");
    expect(initialsElements.length).toBeGreaterThanOrEqual(1);
  });

  // 5. Green status dot when connected with no location error
  it("shows green status dot when connected with no location error", () => {
    mockWebSocket.connectionState = "connected";
    mocks.locationError = null;
    renderLayout();

    const statusContainer = screen.getByTitle("All systems operational");
    expect(statusContainer).toBeInTheDocument();
    const dot = statusContainer.querySelector("span.inline-block");
    expect(dot).toHaveClass("bg-green-500");
  });

  // 6. Redirects to /login when not authenticated
  it("redirects to /login when not authenticated", async () => {
    mockAuth.isAuthenticated = false;
    mockAuth.isLoading = false;
    renderLayout();

    await waitFor(() => {
      expect(mocks.routerPush).toHaveBeenCalledWith("/login");
    });
  });

  // 7. Loading skeleton
  it("shows skeleton loading state when isLoading is true", () => {
    mockAuth.isLoading = true;
    renderLayout();

    const skeleton = document.querySelector('[data-slot="skeleton"]');
    expect(skeleton).toBeInTheDocument();
    // Should NOT render children
    expect(screen.queryByText("page content")).not.toBeInTheDocument();
  });

  // 8. Returns null when not authenticated
  it("renders nothing when not authenticated and not loading", () => {
    mockAuth.isAuthenticated = false;
    mockAuth.isLoading = false;
    const { container } = renderLayout();

    // The layout returns null, so the container should be empty
    // (aside from the wrapping div that render() always creates)
    expect(container.innerHTML).toBe("");
  });

  // 9. No "Server Settings" for non-admin (open dropdown to check)
  it('does not show "Server Settings" link for non-admin user', async () => {
    const user = userEvent.setup();
    mockAuth.isAdmin = false;
    renderLayout();

    // Open the desktop dropdown by clicking the avatar trigger
    const trigger = screen.getByRole("button", { name: /TU/i });
    await user.click(trigger);

    await waitFor(() => {
      expect(screen.getByText("Account Settings")).toBeInTheDocument();
    });
    expect(screen.queryByText("Server Settings")).not.toBeInTheDocument();
  });

  // 10. Shows "Server Settings" for admin (open dropdown to check)
  it('shows "Server Settings" when user is admin', async () => {
    const user = userEvent.setup();
    mockAuth.isAdmin = true;
    renderLayout();

    // Open the desktop dropdown by clicking the avatar trigger
    const trigger = screen.getByRole("button", { name: /TU/i });
    await user.click(trigger);

    await waitFor(() => {
      expect(screen.getByText("Server Settings")).toBeInTheDocument();
    });
  });

  // Bonus: status dot states

  it("shows red status dot when disconnected", () => {
    mockWebSocket.connectionState = "disconnected";
    renderLayout();

    const statusContainer = screen.getByTitle("Cannot connect to server");
    const dot = statusContainer.querySelector("span.inline-block");
    expect(dot).toHaveClass("bg-red-500");
  });

  it("shows yellow status dot when connected but location error exists", () => {
    mockWebSocket.connectionState = "connected";
    mocks.locationError = "Location permission denied";
    renderLayout();

    const statusContainer = screen.getByTitle("Location permission denied");
    const dot = statusContainer.querySelector("span.inline-block");
    expect(dot).toHaveClass("bg-yellow-500");
  });
});
