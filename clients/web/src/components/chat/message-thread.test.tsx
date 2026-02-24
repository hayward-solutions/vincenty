import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import { MessageThread } from "./message-thread";
import type { MessageResponse } from "@/types/api";
import { mockMessage } from "@/test/fixtures";

vi.mock("next/link", () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

// ScrollArea uses ResizeObserver which isn't available in jsdom
// and scrollIntoView is not implemented in jsdom
beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver;

  Element.prototype.scrollIntoView = vi.fn();
});

function makeMessage(
  overrides: Partial<MessageResponse> = {}
): MessageResponse {
  return { ...mockMessage, ...overrides };
}

const msg1 = makeMessage({
  id: "msg-1",
  sender_id: "user-1",
  content: "First message",
  created_at: "2025-01-01T10:00:00Z",
});

const msg2 = makeMessage({
  id: "msg-2",
  sender_id: "user-2",
  username: "other",
  display_name: "Other User",
  content: "Second message",
  created_at: "2025-01-01T10:01:00Z",
});

const msg3 = makeMessage({
  id: "msg-3",
  sender_id: "user-1",
  content: "Third message",
  created_at: "2025-01-01T10:02:00Z",
});

describe("MessageThread", () => {
  describe("empty state", () => {
    it("shows empty state message when there are no messages and not loading", () => {
      render(
        <MessageThread
          messages={[]}
          currentUserId="user-1"
          isLoading={false}
          hasMore={false}
          onLoadMore={vi.fn()}
        />
      );
      expect(
        screen.getByText("No messages yet. Start the conversation!")
      ).toBeInTheDocument();
    });
  });

  describe("message rendering", () => {
    it("renders messages in reversed order (oldest first for display)", () => {
      // API sends newest-first: [msg3, msg2, msg1]
      render(
        <MessageThread
          messages={[msg3, msg2, msg1]}
          currentUserId="user-1"
          isLoading={false}
          hasMore={false}
          onLoadMore={vi.fn()}
        />
      );

      const texts = screen.getAllByText(/message$/i);
      expect(texts[0]).toHaveTextContent("First message");
      expect(texts[1]).toHaveTextContent("Second message");
      expect(texts[2]).toHaveTextContent("Third message");
    });

    it("identifies own messages by sender_id matching currentUserId", () => {
      // msg1 has sender_id "user-1", msg2 has sender_id "user-2"
      const { container } = render(
        <MessageThread
          messages={[msg2, msg1]}
          currentUserId="user-1"
          isLoading={false}
          hasMore={false}
          onLoadMore={vi.fn()}
        />
      );

      // Own messages get ml-auto, others get mr-auto
      const mlAuto = container.querySelectorAll(".ml-auto");
      const mrAuto = container.querySelectorAll(".mr-auto");
      expect(mlAuto.length).toBe(1);
      expect(mrAuto.length).toBe(1);
    });
  });

  describe("load more button", () => {
    it("shows 'Load older messages' button when hasMore=true and isLoading=false", () => {
      render(
        <MessageThread
          messages={[msg1]}
          currentUserId="user-1"
          isLoading={false}
          hasMore={true}
          onLoadMore={vi.fn()}
        />
      );
      expect(
        screen.getByText("Load older messages")
      ).toBeInTheDocument();
    });

    it("shows loading indicator when hasMore=true and isLoading=true", () => {
      const { container } = render(
        <MessageThread
          messages={[msg1]}
          currentUserId="user-1"
          isLoading={true}
          hasMore={true}
          onLoadMore={vi.fn()}
        />
      );
      // Loader2 renders an svg with animate-spin class
      const spinner = container.querySelector(".animate-spin");
      expect(spinner).toBeInTheDocument();
      expect(screen.queryByText("Load older messages")).not.toBeInTheDocument();
    });

    it("shows neither button nor spinner when hasMore=false", () => {
      const { container } = render(
        <MessageThread
          messages={[msg1]}
          currentUserId="user-1"
          isLoading={false}
          hasMore={false}
          onLoadMore={vi.fn()}
        />
      );
      expect(
        screen.queryByText("Load older messages")
      ).not.toBeInTheDocument();
      expect(container.querySelector(".animate-spin")).not.toBeInTheDocument();
    });

    it("calls onLoadMore when 'Load older messages' is clicked", async () => {
      const user = userEvent.setup();
      const onLoadMore = vi.fn();

      render(
        <MessageThread
          messages={[msg1]}
          currentUserId="user-1"
          isLoading={false}
          hasMore={true}
          onLoadMore={onLoadMore}
        />
      );

      await user.click(screen.getByText("Load older messages"));
      expect(onLoadMore).toHaveBeenCalledTimes(1);
    });
  });
});
