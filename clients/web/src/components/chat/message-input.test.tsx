import { screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "@/test/test-utils";
import { MessageInput } from "./message-input";

/** Helper to get the submit (Send) button — it's icon-only with no accessible name. */
function getSendButton() {
  return document.querySelector('button[type="submit"]') as HTMLButtonElement;
}

describe("MessageInput", () => {
  describe("rendering", () => {
    it("renders input with placeholder 'Type a message...'", () => {
      render(<MessageInput onSend={vi.fn()} />);
      expect(
        screen.getByPlaceholderText("Type a message...")
      ).toBeInTheDocument();
    });
  });

  describe("text submission", () => {
    it("calls onSend with trimmed text and empty files when send is clicked", async () => {
      const user = userEvent.setup();
      const onSend = vi.fn();
      render(<MessageInput onSend={onSend} />);

      const input = screen.getByPlaceholderText("Type a message...");
      await user.type(input, "  Hello world  ");
      await user.click(getSendButton());

      expect(onSend).toHaveBeenCalledTimes(1);
      expect(onSend).toHaveBeenCalledWith("Hello world", []);
    });

    it("clears the input after send", async () => {
      const user = userEvent.setup();
      render(<MessageInput onSend={vi.fn()} />);

      const input = screen.getByPlaceholderText("Type a message...");
      await user.type(input, "Hello");
      await user.click(getSendButton());

      expect(input).toHaveValue("");
    });

    it("submits on Enter key (without Shift)", async () => {
      const user = userEvent.setup();
      const onSend = vi.fn();
      render(<MessageInput onSend={onSend} />);

      const input = screen.getByPlaceholderText("Type a message...");
      await user.type(input, "Hello{Enter}");

      expect(onSend).toHaveBeenCalledTimes(1);
      expect(onSend).toHaveBeenCalledWith("Hello", []);
    });

    it("does not call onSend when text is empty (send button disabled)", () => {
      render(<MessageInput onSend={vi.fn()} />);
      expect(getSendButton()).toBeDisabled();
    });
  });

  describe("disabled state", () => {
    it("disables the text input when disabled prop is true", () => {
      render(<MessageInput onSend={vi.fn()} disabled />);
      expect(
        screen.getByPlaceholderText("Type a message...")
      ).toBeDisabled();
    });

    it("disables the send button when disabled prop is true", () => {
      render(<MessageInput onSend={vi.fn()} disabled />);
      expect(getSendButton()).toBeDisabled();
    });

    it("disables the attach button when disabled prop is true", () => {
      render(<MessageInput onSend={vi.fn()} disabled />);
      const buttons = screen.getAllByRole("button");
      // The first button is the attach button (Paperclip)
      const attachButton = buttons[0];
      expect(attachButton).toBeDisabled();
    });
  });

  describe("file handling", () => {
    it("shows file preview chips after selecting files", () => {
      render(<MessageInput onSend={vi.fn()} />);

      const file = new File(["hello"], "hello.txt", { type: "text/plain" });
      const fileInput = document.querySelector(
        'input[type="file"]'
      ) as HTMLInputElement;

      fireEvent.change(fileInput, { target: { files: [file] } });

      expect(screen.getByText("hello.txt")).toBeInTheDocument();
    });

    it("removes a file chip when its remove button is clicked", async () => {
      const user = userEvent.setup();
      render(<MessageInput onSend={vi.fn()} />);

      const file = new File(["content"], "test.pdf", {
        type: "application/pdf",
      });
      const fileInput = document.querySelector(
        'input[type="file"]'
      ) as HTMLInputElement;

      fireEvent.change(fileInput, { target: { files: [file] } });
      expect(screen.getByText("test.pdf")).toBeInTheDocument();

      // The remove button is next to the filename inside the chip
      const removeButton = screen
        .getByText("test.pdf")
        .closest("div")!
        .querySelector("button")!;
      await user.click(removeButton);

      expect(screen.queryByText("test.pdf")).not.toBeInTheDocument();
    });

    it("can submit with files only and no text", async () => {
      const user = userEvent.setup();
      const onSend = vi.fn();
      render(<MessageInput onSend={onSend} />);

      const file = new File(["data"], "image.png", { type: "image/png" });
      const fileInput = document.querySelector(
        'input[type="file"]'
      ) as HTMLInputElement;

      fireEvent.change(fileInput, { target: { files: [file] } });

      // With a file selected, the send button should be enabled
      const sendButton = getSendButton();
      expect(sendButton).not.toBeDisabled();

      await user.click(sendButton);
      expect(onSend).toHaveBeenCalledTimes(1);
      expect(onSend).toHaveBeenCalledWith("", [file]);
    });

    it("alerts when a file exceeds the size limit", () => {
      const alertMock = vi.spyOn(window, "alert").mockImplementation(() => {});
      render(<MessageInput onSend={vi.fn()} />);

      // Create a file object that reports a large size
      const bigFile = new File(["x"], "huge.zip", {
        type: "application/zip",
      });
      Object.defineProperty(bigFile, "size", {
        value: 30 * 1024 * 1024,
      }); // 30 MB

      const fileInput = document.querySelector(
        'input[type="file"]'
      ) as HTMLInputElement;

      fireEvent.change(fileInput, { target: { files: [bigFile] } });

      expect(alertMock).toHaveBeenCalledWith(
        expect.stringContaining("huge.zip")
      );
      // The oversized file should not be in the preview
      expect(screen.queryByText("huge.zip")).not.toBeInTheDocument();

      alertMock.mockRestore();
    });
  });
});
