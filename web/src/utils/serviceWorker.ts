/**
 * Service Worker Registration
 *
 * Registers the service worker for PWA support and offline capability.
 * Works in both development and production environments for testing.
 */

const UPDATE_CHECK_INTERVAL = 60 * 60 * 1000; // 1 hour

// Conditional logging - only in development
const log = {
  info: (...args: unknown[]) => {
    if (import.meta.env.DEV) console.log("[ServiceWorker]", ...args);
  },
  error: (...args: unknown[]) => {
    console.error("[ServiceWorker]", ...args);
  },
};

const registerServiceWorker = () => {
  if (typeof window === "undefined" || !("serviceWorker" in navigator)) {
    log.info("Service Worker not supported in this environment");
    return;
  }

  // Store interval ID for cleanup
  let updateIntervalId: number | undefined;

  window.addEventListener("load", () => {
    const swUrl = "/sw.js";

    navigator.serviceWorker
      .register(swUrl)
      .then((registration) => {
        log.info("Service Worker registered:", registration);

        // Check for updates
        registration.addEventListener("updatefound", () => {
          const newWorker = registration.installing;
          if (newWorker) {
            newWorker.addEventListener("statechange", () => {
              if (newWorker.state === "installed" && navigator.serviceWorker.controller) {
                // New version available
                log.info("New service worker available. Refresh to update.");
                // Dispatch custom event for UI to handle
                window.dispatchEvent(new CustomEvent("sw-update-available"));
              }
            });
          }
        });

        // Periodic update check (every hour)
        updateIntervalId = window.setInterval(() => {
          registration.update();
        }, UPDATE_CHECK_INTERVAL);
      })
      .catch((error) => {
        log.error("Service Worker registration failed:", error);
      });
  });

  // Handle service worker controlling the page
  let refreshing = false;
  navigator.serviceWorker.addEventListener("controllerchange", () => {
    if (!refreshing) {
      refreshing = true;
      window.location.reload();
    }
  });

  // Cleanup interval on page unload to prevent memory leaks
  window.addEventListener("beforeunload", () => {
    if (updateIntervalId !== undefined) {
      clearInterval(updateIntervalId);
    }
  });
};

/**
 * Request the service worker to skip waiting and activate the new version
 */
const skipWaiting = () => {
  if ("serviceWorker" in navigator && navigator.serviceWorker.controller) {
    navigator.serviceWorker.controller.postMessage({ type: "SKIP_WAITING" });
  }
};

/**
 * Clear all caches
 */
const clearCaches = () => {
  if ("serviceWorker" in navigator && navigator.serviceWorker.controller) {
    navigator.serviceWorker.controller.postMessage({ type: "CLEAR_CACHE" });
  }
};

export { registerServiceWorker, skipWaiting, clearCaches };
