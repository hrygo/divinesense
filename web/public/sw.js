/**
 * Service Worker for DivineSense PWA
 *
 * Provides:
 * - Offline caching for static assets
 * - Cache-first strategy for images and fonts
 * - Network-first strategy for API calls and pages
 * - Update notification support
 */

const CACHE_VERSION = "v1";
const CACHE_PREFIX = "divinesense";
const CACHE_NAME = `${CACHE_PREFIX}-${CACHE_VERSION}`;
const STATIC_CACHE = `${CACHE_PREFIX}-static-${CACHE_VERSION}`;
const API_CACHE = `${CACHE_PREFIX}-api-${CACHE_VERSION}`;

// Assets to cache on install
const PRECACHE_URLS = ["/", "/offline"];

// Install event - cache static assets
self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      return cache.addAll(PRECACHE_URLS);
    }),
  );
  // Force activation
  self.skipWaiting();
});

// Activate event - clean up old caches
self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames
          .filter((cacheName) => {
            // Only delete caches that:
            // 1. Start with our prefix
            // 2. Are NOT the current active caches
            return (
              cacheName.startsWith(CACHE_PREFIX) &&
              cacheName !== CACHE_NAME &&
              cacheName !== STATIC_CACHE &&
              cacheName !== API_CACHE
            );
          })
          .map((cacheName) => {
            return caches.delete(cacheName);
          }),
      );
    }),
  );
  // Take control immediately
  self.clients.claim();
});

// Fetch event - routing strategy
self.addEventListener("fetch", (event) => {
  const { request } = event;
  const url = new URL(request.url);

  // Skip non-GET requests
  if (request.method !== "GET") return;

  // Skip external requests
  if (url.origin !== self.location.origin) return;

  // API calls - Network First with cache fallback (for Connect RPC and REST)
  if (url.pathname.startsWith("/api") || url.pathname.startsWith("/demo")) {
    event.respondWith(
      fetch(request)
        .then((response) => {
          // Only cache successful responses
          if (!response.ok) return response;
          // Clone response before caching
          const responseClone = response.clone();
          caches.open(API_CACHE).then((cache) => {
            cache.put(request, responseClone);
          });
          return response;
        })
        .catch(() => {
          return caches.match(request);
        }),
    );
    return;
  }

  // Images and fonts - Cache First
  if (
    request.destination === "image" ||
    request.destination === "font" ||
    url.pathname.match(/\.(jpg|jpeg|png|gif|webp|svg|woff|woff2|ttf|otf|ico)$/)
  ) {
    event.respondWith(
      caches.match(request).then((response) => {
        if (response) {
          return response;
        }
        return fetch(request).then((response) => {
          const responseClone = response.clone();
          caches.open(STATIC_CACHE).then((cache) => {
            cache.put(request, responseClone);
          });
          return response;
        });
      }),
    );
    return;
  }

  // Pages - Network First with cache fallback
  event.respondWith(
    fetch(request)
      .then((response) => {
        // Only cache successful responses
        if (!response.ok) return response;
        const responseClone = response.clone();
        caches.open(CACHE_NAME).then((cache) => {
          cache.put(request, responseClone);
        });
        return response;
      })
      .catch(() => {
        return caches.match(request).then((response) => {
          // Return offline page for navigation requests
          if (request.mode === "navigate" && !response) {
            return caches.match("/offline");
          }
          return response;
        });
      }),
  );
});

// Message event - handle messages from clients
self.addEventListener("message", (event) => {
  if (event.data && event.data.type === "SKIP_WAITING") {
    self.skipWaiting();
  }
  if (event.data && event.data.type === "CLEAR_CACHE") {
    event.waitUntil(
      caches.keys().then((cacheNames) => {
        return Promise.all(
          cacheNames.map((cacheName) => {
            return caches.delete(cacheName);
          }),
        );
      }),
    );
  }
});

// Service Worker file - no ES module exports
