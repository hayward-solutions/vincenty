import { screen } from "@testing-library/react";
import { render, mockAuth } from "@/test/test-utils";
import DashboardPage from "./page";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
  usePathname: () => "/dashboard",
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

describe("DashboardPage", () => {
  it("renders the Dashboard heading", () => {
    render(<DashboardPage />);
    expect(screen.getByText("Dashboard")).toBeInTheDocument();
  });

  it("displays the user display name", () => {
    render(<DashboardPage />);
    expect(screen.getByText("Test User")).toBeInTheDocument();
  });

  it("displays stat card headings", () => {
    render(<DashboardPage />);
    expect(screen.getAllByText("My Groups").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Conversations").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Devices").length).toBeGreaterThan(0);
  });

  it("shows Admin badge when user is admin", () => {
    mockAuth.isAdmin = true;
    mockAuth.user = { ...mockAuth.user, is_admin: true };
    render(<DashboardPage />);
    expect(screen.getByText("Admin")).toBeInTheDocument();
    // Reset
    mockAuth.isAdmin = false;
    mockAuth.user = { ...mockAuth.user, is_admin: false };
  });

  it("does not show Admin badge for non-admin", () => {
    render(<DashboardPage />);
    expect(screen.queryByText("Admin")).not.toBeInTheDocument();
  });

  it("renders the welcome message", () => {
    render(<DashboardPage />);
    expect(screen.getByText("Welcome back,", { exact: false })).toBeInTheDocument();
    expect(screen.getByText("Test User")).toBeInTheDocument();
  });
});
