import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import { SettingsSidebar } from "./settings-sidebar";

vi.mock("next/link", () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

let mockPathname = "/settings/profile";
vi.mock("next/navigation", () => ({
  usePathname: () => mockPathname,
}));

const items = [
  { href: "/settings/profile", label: "Profile" },
  { href: "/settings/devices", label: "Devices" },
  { href: "/settings/security", label: "Security" },
];

function renderSidebar(
  overrides: { title?: string; items?: typeof items; pathname?: string } = {}
) {
  if (overrides.pathname !== undefined) {
    mockPathname = overrides.pathname;
  }
  return render(
    <SettingsSidebar
      title={overrides.title ?? "Settings"}
      items={overrides.items ?? items}
    />
  );
}

afterEach(() => {
  mockPathname = "/settings/profile";
});

describe("SettingsSidebar", () => {
  describe("rendering", () => {
    it("renders the title", () => {
      renderSidebar({ title: "Settings" });
      // Title appears in both desktop and mobile views
      expect(screen.getAllByText("Settings").length).toBeGreaterThanOrEqual(1);
    });

    it("renders all nav items", () => {
      renderSidebar();
      // Each item renders in both desktop nav and mobile sheet nav
      expect(screen.getAllByText("Profile").length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText("Devices").length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText("Security").length).toBeGreaterThanOrEqual(1);
    });
  });

  describe("active state", () => {
    it("active item gets secondary variant", () => {
      renderSidebar({ pathname: "/settings/profile" });
      // Find the desktop nav (md:block)
      const desktopNav = document.querySelector("nav.hidden.md\\:block");
      const buttons = desktopNav?.querySelectorAll("button, a");
      // The Profile link should have data-variant="secondary" (set by Button component)
      // or a secondary class pattern. Check the parent button/link.
      const profileLink = desktopNav?.querySelector(
        'a[href="/settings/profile"]'
      );
      expect(profileLink).toBeInTheDocument();
      // The Button with variant="secondary" will have the data attribute
      expect(profileLink).toHaveAttribute("data-variant", "secondary");
    });

    it("inactive items get ghost variant", () => {
      renderSidebar({ pathname: "/settings/profile" });
      const desktopNav = document.querySelector("nav.hidden.md\\:block");
      const devicesLink = desktopNav?.querySelector(
        'a[href="/settings/devices"]'
      );
      expect(devicesLink).toBeInTheDocument();
      expect(devicesLink).toHaveAttribute("data-variant", "ghost");
    });

    it("sub-path matching: /settings/profile/edit matches /settings/profile", () => {
      renderSidebar({ pathname: "/settings/profile/edit" });
      const desktopNav = document.querySelector("nav.hidden.md\\:block");
      const profileLink = desktopNav?.querySelector(
        'a[href="/settings/profile"]'
      );
      expect(profileLink).toHaveAttribute("data-variant", "secondary");
    });
  });

  describe("desktop nav", () => {
    it("desktop nav element has md:block class", () => {
      renderSidebar();
      const desktopNav = document.querySelector("nav.hidden.md\\:block");
      expect(desktopNav).toBeInTheDocument();
    });
  });

  describe("mobile menu", () => {
    it("has a menu button with sr-only text", () => {
      renderSidebar();
      expect(screen.getByText("Open settings menu")).toBeInTheDocument();
      // The sr-only text is inside a button
      const srText = screen.getByText("Open settings menu");
      expect(srText.closest("button")).toBeInTheDocument();
    });
  });
});
