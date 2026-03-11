import { render } from "@/test/test-utils";
import { ServiceWorkerRegistration } from "./service-worker-registration";

describe("ServiceWorkerRegistration", () => {
  const originalServiceWorker = navigator.serviceWorker;

  afterEach(() => {
    // Restore original serviceWorker property
    Object.defineProperty(navigator, "serviceWorker", {
      value: originalServiceWorker,
      writable: true,
      configurable: true,
    });
  });

  it("renders nothing", () => {
    const { container } = render(<ServiceWorkerRegistration />);
    expect(container.innerHTML).toBe("");
  });

  it("registers service worker when supported", async () => {
    const mockRegister = vi.fn().mockResolvedValue({ scope: "/" });
    Object.defineProperty(navigator, "serviceWorker", {
      value: { register: mockRegister },
      writable: true,
      configurable: true,
    });

    render(<ServiceWorkerRegistration />);

    await vi.waitFor(() => {
      expect(mockRegister).toHaveBeenCalledWith("/sw.js");
    });
  });

  it("does not throw when navigator.serviceWorker is unavailable", () => {
    // Delete serviceWorker so `"serviceWorker" in navigator` returns false
    const desc = Object.getOwnPropertyDescriptor(navigator, "serviceWorker");
    // @ts-expect-error - removing property for test
    delete (navigator as any).serviceWorker;

    expect(() => {
      render(<ServiceWorkerRegistration />);
    }).not.toThrow();

    // Restore if it was defined
    if (desc) {
      Object.defineProperty(navigator, "serviceWorker", desc);
    }
  });
});
