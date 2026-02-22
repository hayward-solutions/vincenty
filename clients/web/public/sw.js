// Service Worker for caching map tiles
// Strategy: Cache-first with LRU eviction at 2000 entries (~200MB)

const CACHE_NAME = "map-tiles-v1";
const MAX_CACHE_SIZE = 2000;

/**
 * Determine whether a request is for a map tile.
 * Matches patterns like:
 *   - /api/v1/tiles/{z}/{x}/{y}.png  (production proxy)
 *   - https://tile.openstreetmap.org/{z}/{x}/{y}.png
 *   - Any URL with /{z}/{x}/{y}.png or /{z}/{x}/{y}.pbf path segments
 *     where z/x/y are numeric
 */
function isTileRequest(url) {
  // Match /{number}/{number}/{number}.{ext} at the end of the pathname
  return /\/\d+\/\d+\/\d+\.(png|jpg|jpeg|webp|pbf|mvt)(\?.*)?$/.test(url.pathname);
}

/**
 * Trim the cache to MAX_CACHE_SIZE by removing the oldest entries.
 * Cache API keys() returns entries in insertion order, so the first
 * entries are the oldest.
 */
async function trimCache(cache) {
  const keys = await cache.keys();
  if (keys.length <= MAX_CACHE_SIZE) return;

  const deleteCount = keys.length - MAX_CACHE_SIZE;
  for (let i = 0; i < deleteCount; i++) {
    await cache.delete(keys[i]);
  }
}

// Install: activate immediately without waiting
self.addEventListener("install", () => {
  self.skipWaiting();
});

// Activate: claim all clients so the SW takes effect immediately
self.addEventListener("activate", (event) => {
  event.waitUntil(self.clients.claim());
});

// Fetch: cache-first for tile requests, passthrough for everything else
self.addEventListener("fetch", (event) => {
  const url = new URL(event.request.url);

  if (!isTileRequest(url)) return;

  event.respondWith(
    (async () => {
      const cache = await caches.open(CACHE_NAME);

      // Check cache first
      const cached = await cache.match(event.request);
      if (cached) return cached;

      // Cache miss: fetch from network
      const response = await fetch(event.request);

      // Only cache successful responses
      if (response.ok) {
        // Clone before consuming — responses can only be read once
        cache.put(event.request, response.clone());

        // Evict oldest entries if cache is too large (non-blocking)
        trimCache(cache);
      }

      return response;
    })()
  );
});
