import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterAll, afterEach, beforeAll, beforeEach } from "vitest";
import { server } from "./msw-server";

// ---------------------------------------------------------------------------
// Node 25 ships a native `localStorage` stub that lacks `.clear()` and other
// methods unless `--localstorage-file` is provided. jsdom's JSDOM instance
// provides its own Storage, but Vitest's jsdom environment doesn't always
// override the global properly. Polyfill a simple in-memory Storage to ensure
// tests work reliably.
// ---------------------------------------------------------------------------

function createMemoryStorage(): Storage {
  let store: Record<string, string> = {};

  return {
    get length() {
      return Object.keys(store).length;
    },
    clear() {
      store = {};
    },
    getItem(key: string) {
      return key in store ? store[key] : null;
    },
    key(index: number) {
      const keys = Object.keys(store);
      return keys[index] ?? null;
    },
    removeItem(key: string) {
      delete store[key];
    },
    setItem(key: string, value: string) {
      store[key] = String(value);
    },
  };
}

// Only polyfill if the native localStorage is broken (no .clear method)
if (typeof globalThis.localStorage?.clear !== "function") {
  const memStorage = createMemoryStorage();
  Object.defineProperty(globalThis, "localStorage", {
    value: memStorage,
    writable: true,
    configurable: true,
  });
}

// Start MSW server before all tests
beforeAll(() => server.listen({ onUnhandledRequest: "error" }));

// Reset handlers and clean up DOM after each test
afterEach(() => {
  server.resetHandlers();
  cleanup();
  localStorage.clear();
});

// Stop the MSW server after all tests
afterAll(() => server.close());

// Mock requestAnimationFrame for tests (used by use-locations.ts)
if (typeof globalThis.requestAnimationFrame === "undefined") {
  globalThis.requestAnimationFrame = (cb: FrameRequestCallback) => {
    return setTimeout(() => cb(Date.now()), 0) as unknown as number;
  };
  globalThis.cancelAnimationFrame = (id: number) => {
    clearTimeout(id);
  };
}
