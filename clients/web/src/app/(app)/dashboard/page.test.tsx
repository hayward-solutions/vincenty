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

  it("displays the user email", () => {
    render(<DashboardPage />);
    expect(screen.getByText("test@example.com")).toBeInTheDocument();
  });

  it("shows Admin badge when user is admin", () => {
    mockAuth.user = { ...mockAuth.user, is_admin: true };
    render(<DashboardPage />);
    expect(screen.getByText("Admin")).toBeInTheDocument();
    // Reset
    mockAuth.user = { ...mockAuth.user, is_admin: false };
  });

  it("does not show Admin badge for non-admin", () => {
    render(<DashboardPage />);
    expect(screen.queryByText("Admin")).not.toBeInTheDocument();
  });

  it("renders Map and Messages info cards", () => {
    render(<DashboardPage />);
    expect(screen.getByText("Map")).toBeInTheDocument();
    expect(screen.getByText("Messages")).toBeInTheDocument();
  });
});
