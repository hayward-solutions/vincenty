import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import { ConversationList } from "./conversation-list";
import type { Conversation } from "@/types/api";

// ScrollArea uses ResizeObserver which isn't available in jsdom
beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver;
});

const groupConversation: Conversation = {
  id: "group-1",
  type: "group",
  name: "Alpha Team",
};

const groupConversation2: Conversation = {
  id: "group-2",
  type: "group",
  name: "Bravo Team",
};

const dmConversation: Conversation = {
  id: "user-2",
  type: "direct",
  name: "Jane Doe",
};

const dmConversation2: Conversation = {
  id: "user-3",
  type: "direct",
  name: "John Smith",
};

describe("ConversationList", () => {
  describe("empty state", () => {
    it("shows 'No conversations yet' when conversations is empty", () => {
      render(
        <ConversationList
          conversations={[]}
          activeId={null}
          onSelect={vi.fn()}
        />
      );
      expect(screen.getByText("No conversations yet")).toBeInTheDocument();
    });

    it("does not show section headers when empty", () => {
      render(
        <ConversationList
          conversations={[]}
          activeId={null}
          onSelect={vi.fn()}
        />
      );
      expect(screen.queryByText("Groups")).not.toBeInTheDocument();
      expect(screen.queryByText("Direct Messages")).not.toBeInTheDocument();
    });
  });

  describe("section headers", () => {
    it("renders Groups header when groups exist", () => {
      render(
        <ConversationList
          conversations={[groupConversation]}
          activeId={null}
          onSelect={vi.fn()}
        />
      );
      expect(screen.getByText("Groups")).toBeInTheDocument();
    });

    it("does not render Groups header when only DMs exist", () => {
      render(
        <ConversationList
          conversations={[dmConversation]}
          activeId={null}
          onSelect={vi.fn()}
        />
      );
      expect(screen.queryByText("Groups")).not.toBeInTheDocument();
    });

    it("renders Direct Messages header when DMs exist", () => {
      render(
        <ConversationList
          conversations={[dmConversation]}
          activeId={null}
          onSelect={vi.fn()}
        />
      );
      expect(screen.getByText("Direct Messages")).toBeInTheDocument();
    });

    it("does not render Direct Messages header when only groups exist", () => {
      render(
        <ConversationList
          conversations={[groupConversation]}
          activeId={null}
          onSelect={vi.fn()}
        />
      );
      expect(screen.queryByText("Direct Messages")).not.toBeInTheDocument();
    });

    it("renders both headers when both types exist", () => {
      render(
        <ConversationList
          conversations={[groupConversation, dmConversation]}
          activeId={null}
          onSelect={vi.fn()}
        />
      );
      expect(screen.getByText("Groups")).toBeInTheDocument();
      expect(screen.getByText("Direct Messages")).toBeInTheDocument();
    });
  });

  describe("conversation items", () => {
    it("renders conversation names", () => {
      render(
        <ConversationList
          conversations={[groupConversation, dmConversation]}
          activeId={null}
          onSelect={vi.fn()}
        />
      );
      expect(screen.getByText("Alpha Team")).toBeInTheDocument();
      expect(screen.getByText("Jane Doe")).toBeInTheDocument();
    });

    it("renders multiple groups and DMs", () => {
      render(
        <ConversationList
          conversations={[
            groupConversation,
            groupConversation2,
            dmConversation,
            dmConversation2,
          ]}
          activeId={null}
          onSelect={vi.fn()}
        />
      );
      expect(screen.getByText("Alpha Team")).toBeInTheDocument();
      expect(screen.getByText("Bravo Team")).toBeInTheDocument();
      expect(screen.getByText("Jane Doe")).toBeInTheDocument();
      expect(screen.getByText("John Smith")).toBeInTheDocument();
    });
  });

  describe("active state", () => {
    it("applies bg-accent class to the active conversation", () => {
      render(
        <ConversationList
          conversations={[groupConversation, dmConversation]}
          activeId="group-1"
          onSelect={vi.fn()}
        />
      );
      const activeButton = screen.getByText("Alpha Team").closest("button");
      // Check for exact "bg-accent" class token (not hover:bg-accent)
      const classes = activeButton?.className.split(/\s+/) ?? [];
      expect(classes).toContain("bg-accent");
    });

    it("does not apply bg-accent to inactive conversations", () => {
      render(
        <ConversationList
          conversations={[groupConversation, dmConversation]}
          activeId="group-1"
          onSelect={vi.fn()}
        />
      );
      const inactiveButton = screen.getByText("Jane Doe").closest("button");
      // Should not have the exact "bg-accent" token (hover:bg-accent is fine)
      const classes = inactiveButton?.className.split(/\s+/) ?? [];
      expect(classes).not.toContain("bg-accent");
    });
  });

  describe("onSelect", () => {
    it("calls onSelect with the conversation when clicked", async () => {
      const user = userEvent.setup();
      const onSelect = vi.fn();

      render(
        <ConversationList
          conversations={[groupConversation, dmConversation]}
          activeId={null}
          onSelect={onSelect}
        />
      );

      await user.click(screen.getByText("Alpha Team"));
      expect(onSelect).toHaveBeenCalledTimes(1);
      expect(onSelect).toHaveBeenCalledWith(groupConversation);
    });

    it("calls onSelect with the correct DM conversation", async () => {
      const user = userEvent.setup();
      const onSelect = vi.fn();

      render(
        <ConversationList
          conversations={[groupConversation, dmConversation]}
          activeId={null}
          onSelect={onSelect}
        />
      );

      await user.click(screen.getByText("Jane Doe"));
      expect(onSelect).toHaveBeenCalledWith(dmConversation);
    });
  });

  describe("New Message button", () => {
    it("shows New Message button when onNewMessage is provided", () => {
      render(
        <ConversationList
          conversations={[]}
          activeId={null}
          onSelect={vi.fn()}
          onNewMessage={vi.fn()}
        />
      );
      expect(
        screen.getByRole("button", { name: /new message/i })
      ).toBeInTheDocument();
    });

    it("hides New Message button when onNewMessage is not provided", () => {
      render(
        <ConversationList
          conversations={[]}
          activeId={null}
          onSelect={vi.fn()}
        />
      );
      expect(
        screen.queryByRole("button", { name: /new message/i })
      ).not.toBeInTheDocument();
    });

    it("calls onNewMessage when New Message button is clicked", async () => {
      const user = userEvent.setup();
      const onNewMessage = vi.fn();

      render(
        <ConversationList
          conversations={[]}
          activeId={null}
          onSelect={vi.fn()}
          onNewMessage={onNewMessage}
        />
      );

      await user.click(screen.getByRole("button", { name: /new message/i }));
      expect(onNewMessage).toHaveBeenCalledTimes(1);
    });
  });
});
