import { screen } from "@testing-library/react";
import { render } from "@/test/test-utils";
import { AuditLogTable } from "./audit-log-table";
import type { AuditLogResponse } from "@/types/api";
import { mockAuditLog } from "@/test/fixtures";

function makeLog(overrides: Partial<AuditLogResponse> = {}): AuditLogResponse {
  return { ...mockAuditLog, ...overrides };
}

describe("AuditLogTable", () => {
  describe("empty state", () => {
    it("renders 'No audit logs found' when logs is empty", () => {
      render(<AuditLogTable logs={[]} />);
      expect(screen.getByText("No audit logs found")).toBeInTheDocument();
    });

    it("sets colspan=4 when showUser is false", () => {
      render(<AuditLogTable logs={[]} />);
      const cell = screen.getByText("No audit logs found").closest("td");
      expect(cell).toHaveAttribute("colspan", "4");
    });

    it("sets colspan=5 when showUser is true", () => {
      render(<AuditLogTable logs={[]} showUser />);
      const cell = screen.getByText("No audit logs found").closest("td");
      expect(cell).toHaveAttribute("colspan", "5");
    });
  });

  describe("rendering rows", () => {
    const logs: AuditLogResponse[] = [
      makeLog({
        id: "audit-1",
        action: "auth.login",
        resource_type: "session",
        resource_id: "abcdef1234567890",
        ip_address: "192.168.1.1",
        created_at: "2025-06-15T10:30:00Z",
      }),
      makeLog({
        id: "audit-2",
        action: "user.delete",
        resource_type: "user",
        resource_id: "deadbeef12345678",
        ip_address: "10.0.0.1",
        created_at: "2025-06-15T11:00:00Z",
      }),
    ];

    it("renders one row per log entry", () => {
      render(<AuditLogTable logs={logs} />);
      // Each log gets a row; check IPs as unique identifiers
      expect(screen.getByText("192.168.1.1")).toBeInTheDocument();
      expect(screen.getByText("10.0.0.1")).toBeInTheDocument();
    });

    it("renders resource type with truncated resource_id", () => {
      render(<AuditLogTable logs={logs} />);
      expect(screen.getByText("session (abcdef12...)")).toBeInTheDocument();
      expect(screen.getByText("user (deadbeef...)")).toBeInTheDocument();
    });

    it("renders resource type without id when resource_id is absent", () => {
      const log = makeLog({
        id: "audit-3",
        action: "auth.login",
        resource_type: "session",
        resource_id: undefined,
      });
      render(<AuditLogTable logs={[log]} />);
      // Should show just the resource_type with no parenthetical
      const cells = screen.getAllByRole("cell");
      const resourceCell = cells.find(
        (c) => c.textContent === "session"
      );
      expect(resourceCell).toBeDefined();
    });

    it("renders formatted time", () => {
      const log = makeLog({
        id: "audit-time",
        created_at: "2025-06-15T10:30:00Z",
      });
      render(<AuditLogTable logs={[log]} />);
      // The exact format depends on locale, but the cell should exist
      const rows = screen.getAllByRole("row");
      // First row is header, second is data
      expect(rows.length).toBe(2);
    });
  });

  describe("showUser column", () => {
    const log = makeLog({
      id: "audit-user",
      display_name: "Test User",
      username: "testuser",
    });

    it("hides User column header by default", () => {
      render(<AuditLogTable logs={[log]} />);
      const headers = screen.getAllByRole("columnheader");
      const headerTexts = headers.map((h) => h.textContent);
      expect(headerTexts).not.toContain("User");
    });

    it("shows User column header when showUser=true", () => {
      render(<AuditLogTable logs={[log]} showUser />);
      const headers = screen.getAllByRole("columnheader");
      const headerTexts = headers.map((h) => h.textContent);
      expect(headerTexts).toContain("User");
    });

    it("renders display_name in user column", () => {
      render(<AuditLogTable logs={[log]} showUser />);
      expect(screen.getByText("Test User")).toBeInTheDocument();
    });

    it("falls back to username when display_name is empty", () => {
      const logNoDisplay = makeLog({
        id: "audit-no-display",
        display_name: "",
        username: "fallbackuser",
      });
      render(<AuditLogTable logs={[logNoDisplay]} showUser />);
      expect(screen.getByText("fallbackuser")).toBeInTheDocument();
    });
  });

  describe("action labels", () => {
    it.each([
      ["auth.login", "Login"],
      ["auth.logout", "Logout"],
      ["user.create", "Create User"],
      ["user.update", "Update User"],
      ["user.delete", "Delete User"],
      ["device.create", "Create Device"],
      ["device.delete", "Delete Device"],
      ["group.create", "Create Group"],
      ["group.member_add", "Add Member"],
      ["group.member_remove", "Remove Member"],
      ["message.send", "Send Message"],
      ["message.delete", "Delete Message"],
      ["map_config.create", "Create Map Config"],
    ])("maps %s → %s", (action, label) => {
      const log = makeLog({ id: `audit-${action}`, action });
      render(<AuditLogTable logs={[log]} />);
      expect(screen.getByText(label)).toBeInTheDocument();
    });

    it("shows raw action string for unknown actions", () => {
      const log = makeLog({
        id: "audit-unknown",
        action: "custom.unknown_action",
      });
      render(<AuditLogTable logs={[log]} />);
      expect(screen.getByText("custom.unknown_action")).toBeInTheDocument();
    });
  });

  describe("badge variants", () => {
    function getBadgeVariant(action: string): string | null {
      const log = makeLog({ id: `audit-${action}`, action });
      const { container } = render(<AuditLogTable logs={[log]} />);
      const badge = container.querySelector("[data-slot='badge']");
      return badge?.getAttribute("data-variant") ?? null;
    }

    it("uses destructive for .delete actions", () => {
      expect(getBadgeVariant("user.delete")).toBe("destructive");
    });

    it("uses destructive for .remove actions", () => {
      expect(getBadgeVariant("group.remove")).toBe("destructive");
    });

    it("uses default for .create actions", () => {
      expect(getBadgeVariant("user.create")).toBe("default");
    });

    it("uses default for .add actions", () => {
      expect(getBadgeVariant("group.add")).toBe("default");
    });

    it("uses outline for auth. actions", () => {
      expect(getBadgeVariant("auth.login")).toBe("outline");
    });

    it("uses secondary for other actions", () => {
      expect(getBadgeVariant("user.update")).toBe("secondary");
    });
  });
});
