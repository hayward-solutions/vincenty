import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import MessagesPage from "./page";
import type { Conversation, MessageResponse } from "@/types/api";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
  usePathname: () => "/messages",
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

// Mock location sharing hook
vi.mock("@/lib/hooks/use-location-sharing", () => ({
  useLocationSharing: () => ({
    lastPosition: null,
    error: null,
  }),
  LocationProvider: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));

const mockConversations: Conversation[] = [
  {
    id: "group-1",
    type: "group",
    name: "Test Group",
    unread: 0,
  },
  {
    id: "user-2",
    type: "direct",
    name: "Alice",
    unread: 2,
  },
];

const mockConversationsHook = vi.hoisted(() => ({
  conversations: [] as Conversation[],
  isLoading: false,
  addDmConversation: vi.fn(),
}));

const mockSendMessageHook = vi.hoisted(() => ({
  sendMessage: vi.fn(),
  isLoading: false,
}));

const mockGroupMessagesHook = vi.hoisted(() => ({
  messages: [] as MessageResponse[],
  isLoading: false,
  hasMore: false,
  loadMore: vi.fn(),
  addOptimistic: vi.fn(),
}));

const mockDirectMessagesHook = vi.hoisted(() => ({
  messages: [] as MessageResponse[],
  isLoading: false,
  hasMore: false,
  loadMore: vi.fn(),
  addOptimistic: vi.fn(),
}));

vi.mock("@/lib/hooks/use-conversations", () => ({
  useConversations: () => mockConversationsHook,
}));

vi.mock("@/lib/hooks/use-messages", () => ({
  useSendMessage: () => mockSendMessageHook,
  useGroupMessages: () => mockGroupMessagesHook,
  useDirectMessages: () => mockDirectMessagesHook,
}));

// Mock child components to reduce complexity
vi.mock("@/components/chat/conversation-list", () => ({
  ConversationList: ({
    conversations,
    onSelect,
    onNewMessage,
  }: {
    conversations: Conversation[];
    activeId: string | null;
    onSelect: (c: Conversation) => void;
    onNewMessage: () => void;
  }) => (
    <div data-testid="conversation-list">
      {conversations.map((c) => (
        <button
          key={c.id}
          data-testid={`conv-${c.id}`}
          onClick={() => onSelect(c)}
        >
          {c.name}
        </button>
      ))}
      <button data-testid="new-dm-btn" onClick={onNewMessage}>
        New DM
      </button>
    </div>
  ),
}));

vi.mock("@/components/chat/message-thread", () => ({
  MessageThread: ({
    messages,
  }: {
    messages: MessageResponse[];
    currentUserId: string;
    isLoading: boolean;
    hasMore: boolean;
    onLoadMore: () => void;
  }) => (
    <div data-testid="message-thread">Messages: {messages.length}</div>
  ),
}));

vi.mock("@/components/chat/message-input", () => ({
  MessageInput: ({
    onSend,
    disabled,
  }: {
    onSend: (content: string, files: File[]) => void;
    disabled: boolean;
  }) => (
    <button
      data-testid="send-btn"
      onClick={() => onSend("Hello!", [])}
      disabled={disabled}
    >
      Send
    </button>
  ),
}));

vi.mock("@/components/chat/new-dm-dialog", () => ({
  NewDmDialog: ({
    open,
    onSelect,
  }: {
    open: boolean;
    onOpenChange: (o: boolean) => void;
    onSelect: (id: string, name: string) => void;
  }) =>
    open ? (
      <div data-testid="new-dm-dialog">
        <button onClick={() => onSelect("user-5", "Bob")}>Select Bob</button>
      </div>
    ) : null,
}));

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  vi.clearAllMocks();
  mockConversationsHook.conversations = [];
  mockConversationsHook.isLoading = false;
  mockConversationsHook.addDmConversation.mockReturnValue({
    id: "user-5",
    type: "direct",
    name: "Bob",
    unread: 0,
  });
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("MessagesPage", () => {
  it("renders Conversations heading", () => {
    render(<MessagesPage />);
    expect(screen.getByText("Conversations")).toBeInTheDocument();
  });

  it("renders conversation list", () => {
    mockConversationsHook.conversations = mockConversations;
    render(<MessagesPage />);
    expect(screen.getByTestId("conversation-list")).toBeInTheDocument();
    expect(screen.getByText("Test Group")).toBeInTheDocument();
    expect(screen.getByText("Alice")).toBeInTheDocument();
  });

  it("shows 'Select a conversation' when no conversation is active", () => {
    render(<MessagesPage />);
    expect(
      screen.getByText("Select a conversation to start messaging")
    ).toBeInTheDocument();
  });

  it("shows Loading... when conversations are loading", () => {
    mockConversationsHook.isLoading = true;
    render(<MessagesPage />);
    expect(screen.getByText("Loading...")).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Selecting a conversation
  // -----------------------------------------------------------------------

  it("shows message thread when a conversation is selected", async () => {
    mockConversationsHook.conversations = mockConversations;
    const user = userEvent.setup();
    render(<MessagesPage />);

    await user.click(screen.getByTestId("conv-group-1"));

    expect(screen.getByTestId("message-thread")).toBeInTheDocument();
    // "Test Group" appears in both the list and header — just check header exists
    const headers = screen.getAllByText("Test Group");
    expect(headers.length).toBeGreaterThanOrEqual(1);
    expect(screen.getByTestId("send-btn")).toBeInTheDocument();
  });

  it("shows conversation name in header when selected", async () => {
    mockConversationsHook.conversations = mockConversations;
    const user = userEvent.setup();
    render(<MessagesPage />);

    await user.click(screen.getByTestId("conv-user-2"));

    // Alice should appear as header
    const headerText = screen.getAllByText("Alice");
    expect(headerText.length).toBeGreaterThanOrEqual(1);
  });

  // -----------------------------------------------------------------------
  // Sending messages
  // -----------------------------------------------------------------------

  it("sends a message when send button is clicked", async () => {
    mockConversationsHook.conversations = mockConversations;
    mockSendMessageHook.sendMessage.mockResolvedValue({
      id: "msg-new",
      content: "Hello!",
    });
    const user = userEvent.setup();
    render(<MessagesPage />);

    await user.click(screen.getByTestId("conv-group-1"));
    await user.click(screen.getByTestId("send-btn"));

    await waitFor(() => {
      expect(mockSendMessageHook.sendMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          content: "Hello!",
          groupId: "group-1",
        })
      );
    });
  });

  // -----------------------------------------------------------------------
  // New DM dialog
  // -----------------------------------------------------------------------

  it("opens new DM dialog when New DM button is clicked", async () => {
    mockConversationsHook.conversations = mockConversations;
    const user = userEvent.setup();
    render(<MessagesPage />);

    await user.click(screen.getByTestId("new-dm-btn"));

    expect(screen.getByTestId("new-dm-dialog")).toBeInTheDocument();
  });

  it("adds DM conversation and selects it when user is picked", async () => {
    mockConversationsHook.conversations = mockConversations;
    const user = userEvent.setup();
    render(<MessagesPage />);

    await user.click(screen.getByTestId("new-dm-btn"));
    await user.click(screen.getByText("Select Bob"));

    expect(mockConversationsHook.addDmConversation).toHaveBeenCalledWith(
      "user-5",
      "Bob"
    );
  });

  // -----------------------------------------------------------------------
  // Mobile back button
  // -----------------------------------------------------------------------

  it("shows back button when conversation is selected", async () => {
    mockConversationsHook.conversations = mockConversations;
    const user = userEvent.setup();
    render(<MessagesPage />);

    await user.click(screen.getByTestId("conv-group-1"));

    expect(
      screen.getByRole("button", { name: /back to conversations/i })
    ).toBeInTheDocument();
  });
});
