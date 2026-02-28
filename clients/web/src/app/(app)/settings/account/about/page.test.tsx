import { screen, waitFor } from "@testing-library/react";
import { render } from "@/test/test-utils";
import AboutSettingsPage from "./page";

describe("AboutSettingsPage", () => {
  it('renders heading "About"', () => {
    render(<AboutSettingsPage />);
    expect(screen.getByRole("heading", { name: "About" })).toBeInTheDocument();
  });

  it('renders "Version Information" card', () => {
    render(<AboutSettingsPage />);
    expect(screen.getByText("Version Information")).toBeInTheDocument();
  });

  it("shows Web Client label", () => {
    render(<AboutSettingsPage />);
    expect(screen.getByText("Web Client")).toBeInTheDocument();
  });

  it("shows API Server label", () => {
    render(<AboutSettingsPage />);
    expect(screen.getByText("API Server")).toBeInTheDocument();
  });

  it("fetches and displays the API version", async () => {
    render(<AboutSettingsPage />);
    // The MSW handler returns "dev" for the API version
    await waitFor(() => {
      expect(screen.getByText("dev")).toBeInTheDocument();
    });
  });
});
