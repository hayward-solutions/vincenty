import { screen } from "@testing-library/react";
import { render } from "@/test/test-utils";
import { MessageBubble } from "./message-bubble";
import type { MessageResponse } from "@/types/api";
import { mockMessage } from "@/test/fixtures";

vi.mock("next/link", () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

function makeMessage(
  overrides: Partial<MessageResponse> = {}
): MessageResponse {
  return { ...mockMessage, ...overrides };
}

describe("MessageBubble", () => {
  describe("own message", () => {
    it("aligns to the right with ml-auto", () => {
      const { container } = render(
        <MessageBubble message={mockMessage} isOwn={true} />
      );
      const wrapper = container.firstElementChild as HTMLElement;
      const classes = wrapper.className.split(/\s+/);
      expect(classes).toContain("ml-auto");
    });

    it("uses bg-primary styling", () => {
      const { container } = render(
        <MessageBubble message={mockMessage} isOwn={true} />
      );
      const bubble = container.querySelector(".bg-primary");
      expect(bubble).toBeInTheDocument();
    });

    it("does not show sender name", () => {
      const msg = makeMessage({ display_name: "Test User" });
      render(<MessageBubble message={msg} isOwn={true} />);
      // The sender name is shown in a span before the bubble, not inside it
      const spans = screen.queryAllByText("Test User");
      // The name may appear in the info popover but NOT as the sender label
      // The sender label is only rendered when !isOwn
      const senderLabel = document.querySelector(
        ".text-xs.text-muted-foreground.px-1"
      );
      // When isOwn, the first child of the outer div should be the bubble, not a sender span
      const wrapper = document.querySelector(".ml-auto");
      const firstChild = wrapper?.firstElementChild;
      // First child should be the bubble div, not a <span> with the sender name
      expect(firstChild?.tagName).not.toBe("SPAN");
    });
  });

  describe("other user message", () => {
    it("aligns to the left with mr-auto", () => {
      const { container } = render(
        <MessageBubble message={mockMessage} isOwn={false} />
      );
      const wrapper = container.firstElementChild as HTMLElement;
      const classes = wrapper.className.split(/\s+/);
      expect(classes).toContain("mr-auto");
    });

    it("uses bg-muted styling", () => {
      const { container } = render(
        <MessageBubble message={mockMessage} isOwn={false} />
      );
      const bubble = container.querySelector(".bg-muted");
      expect(bubble).toBeInTheDocument();
    });

    it("shows display_name as sender name", () => {
      const msg = makeMessage({ display_name: "Alice", username: "alice" });
      render(<MessageBubble message={msg} isOwn={false} />);
      // Sender label is in a span before the bubble
      expect(screen.getByText("Alice")).toBeInTheDocument();
    });

    it("falls back to username when display_name is empty", () => {
      const msg = makeMessage({ display_name: "", username: "bob" });
      render(<MessageBubble message={msg} isOwn={false} />);
      expect(screen.getByText("bob")).toBeInTheDocument();
    });
  });

  describe("text content", () => {
    it("renders the message content", () => {
      const msg = makeMessage({ content: "Hello, world!" });
      render(<MessageBubble message={msg} isOwn={false} />);
      expect(screen.getByText("Hello, world!")).toBeInTheDocument();
    });

    it("does not render a text paragraph when content is empty", () => {
      const msg = makeMessage({
        content: "",
        attachments: [
          {
            id: "att-1",
            filename: "doc.pdf",
            content_type: "application/pdf",
            size_bytes: 1024,
            created_at: "2025-01-01T00:00:00Z",
          },
        ],
      });
      render(<MessageBubble message={msg} isOwn={false} />);
      // There should be no <p> with whitespace-pre-wrap
      const paragraphs = document.querySelectorAll("p.whitespace-pre-wrap");
      expect(paragraphs.length).toBe(0);
    });
  });

  describe("image attachment", () => {
    it("renders an img tag with correct alt text", () => {
      const msg = makeMessage({
        attachments: [
          {
            id: "att-img",
            filename: "photo.jpg",
            content_type: "image/jpeg",
            size_bytes: 50000,
            created_at: "2025-01-01T00:00:00Z",
          },
        ],
      });
      render(<MessageBubble message={msg} isOwn={false} />);
      const img = screen.getByRole("img", { name: "photo.jpg" });
      expect(img).toBeInTheDocument();
      expect(img.getAttribute("src")).toContain("/api/v1/attachments/att-img/download");
    });
  });

  describe("file attachment", () => {
    it("renders a download link with filename and file size", () => {
      const msg = makeMessage({
        attachments: [
          {
            id: "att-file",
            filename: "report.pdf",
            content_type: "application/pdf",
            size_bytes: 2048,
            created_at: "2025-01-01T00:00:00Z",
          },
        ],
      });
      render(<MessageBubble message={msg} isOwn={false} />);
      expect(screen.getByText("report.pdf")).toBeInTheDocument();
      expect(screen.getByText("2.0 KB")).toBeInTheDocument();
    });
  });

  describe("GPX message", () => {
    it("shows 'View GPX on Map' link pointing to /map?gpx=<id>", () => {
      const msg = makeMessage({
        id: "msg-gpx-1",
        message_type: "gpx",
        metadata: {},
      });
      render(<MessageBubble message={msg} isOwn={false} />);
      const link = screen.getByText("View GPX on Map");
      expect(link).toBeInTheDocument();
      expect(link.closest("a")).toHaveAttribute("href", "/map?gpx=msg-gpx-1");
    });
  });

  describe("drawing message", () => {
    it("shows 'View Drawing on Map' link pointing to /map?drawing=<drawingId>", () => {
      const msg = makeMessage({
        message_type: "drawing",
        metadata: { drawing_id: "draw-42" },
      });
      render(<MessageBubble message={msg} isOwn={false} />);
      const link = screen.getByText("View Drawing on Map");
      expect(link).toBeInTheDocument();
      expect(link.closest("a")).toHaveAttribute(
        "href",
        "/map?drawing=draw-42"
      );
    });
  });

  describe("footer", () => {
    it("displays time", () => {
      const msg = makeMessage({ created_at: "2025-06-15T14:30:00Z" });
      render(<MessageBubble message={msg} isOwn={false} />);
      // Time is formatted via toLocaleTimeString; just check the footer exists
      const footer = document.querySelector(
        ".text-xs.text-muted-foreground.px-1"
      );
      expect(footer).toBeInTheDocument();
      expect(footer!.textContent).not.toBe("");
    });

    it("shows location coordinates when lat/lng are present", () => {
      const msg = makeMessage({ lat: -33.8688, lng: 151.2093 });
      render(<MessageBubble message={msg} isOwn={false} />);
      // Coordinates formatted to 4 decimal places in the footer
      expect(screen.getByText(/-33\.8688/)).toBeInTheDocument();
      expect(screen.getByText(/151\.2093/)).toBeInTheDocument();
    });

    it("does not show location when lat/lng are absent", () => {
      const msg = makeMessage({ lat: undefined, lng: undefined });
      render(<MessageBubble message={msg} isOwn={false} />);
      // No MapPin text with coordinates
      const footer = document.querySelector(
        ".flex.items-center.gap-1\\.5.text-xs"
      );
      expect(footer?.textContent).not.toMatch(/\d+\.\d{4}/);
    });
  });
});
