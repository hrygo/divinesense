import { timestampDate } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError, createClient, type Interceptor } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { getAccessToken, setAccessToken } from "./auth-state";
import { ActivityService } from "./types/proto/api/v1/activity_service_pb";
import { AIService } from "./types/proto/api/v1/ai_service_pb";
import { AttachmentService } from "./types/proto/api/v1/attachment_service_pb";
import { AuthService } from "./types/proto/api/v1/auth_service_pb";
import { IdentityProviderService } from "./types/proto/api/v1/idp_service_pb";
import { InstanceService } from "./types/proto/api/v1/instance_service_pb";
import { MemoService } from "./types/proto/api/v1/memo_service_pb";
import { ScheduleService } from "./types/proto/api/v1/schedule_service_pb";
import { ShortcutService } from "./types/proto/api/v1/shortcut_service_pb";
import { UserService } from "./types/proto/api/v1/user_service_pb";
import { redirectOnAuthFailure } from "./utils/auth-redirect";

// ============================================================================
// Constants
// ============================================================================

const RETRY_HEADER = "X-Retry";
const RETRY_HEADER_VALUE = "true";

// Default timeout for streaming requests (5 minutes)
// Streaming requests may take longer due to LLM processing time
const DEFAULT_STREAM_TIMEOUT_MS = 10 * 60 * 1000;

// ============================================================================
// Token Refresh State Management
// ============================================================================

const createTokenRefreshManager = () => {
  let isRefreshing = false;
  let refreshPromise: Promise<void> | null = null;

  return {
    async refresh(refreshFn: () => Promise<void>): Promise<void> {
      if (isRefreshing && refreshPromise) {
        return refreshPromise;
      }

      isRefreshing = true;
      refreshPromise = refreshFn().finally(() => {
        isRefreshing = false;
        refreshPromise = null;
      });

      return refreshPromise;
    },
  };
};

const tokenRefreshManager = createTokenRefreshManager();

// ============================================================================
// Token Refresh
// ============================================================================

const fetchWithCredentials: typeof globalThis.fetch = (input, init) => {
  return globalThis.fetch(input, {
    ...init,
    credentials: "include",
  });
};

// Separate transport without auth interceptor to prevent recursion
const refreshTransport = createConnectTransport({
  baseUrl: window.location.origin,
  useBinaryFormat: true,
  fetch: fetchWithCredentials,
  interceptors: [],
});

const refreshAuthClient = createClient(AuthService, refreshTransport);

async function refreshAccessToken(): Promise<void> {
  const response = await refreshAuthClient.refreshToken({});

  if (!response.accessToken) {
    throw new ConnectError("Refresh token response missing access token", Code.Internal);
  }

  const expiresAt = response.expiresAt ? timestampDate(response.expiresAt) : undefined;
  setAccessToken(response.accessToken, expiresAt);
}

// ============================================================================
// Timeout Interceptor
// ============================================================================

// Create a timeout-enabled fetch wrapper
const createTimeoutFetch = (baseFetch: typeof fetch, timeoutMs: number): typeof fetch => {
  return async (input, init) => {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

    try {
      // Pass the abort signal to fetch
      const response = await baseFetch(input, {
        ...init,
        signal: controller.signal,
      });
      clearTimeout(timeoutId);
      return response;
    } catch (error) {
      clearTimeout(timeoutId);
      // Check if it's an abort error (timeout)
      if (error instanceof Error && error.name === "AbortError") {
        throw new Error(`Request timeout after ${timeoutMs}ms`);
      }
      throw error;
    }
  };
};

// The timeout interceptor is minimal since timeout is handled in the fetch wrapper
const timeoutInterceptor: Interceptor = (next) => async (req) => next(req);

// ============================================================================
// Authentication Interceptor
// ============================================================================

const authInterceptor: Interceptor = (next) => async (req) => {
  const token = getAccessToken();
  if (token) {
    req.header.set("Authorization", `Bearer ${token}`);
  }

  try {
    return await next(req);
  } catch (error) {
    if (!(error instanceof ConnectError)) {
      throw error;
    }

    if (error.code !== Code.Unauthenticated) {
      throw error;
    }

    if (req.header.get(RETRY_HEADER) === RETRY_HEADER_VALUE) {
      throw error;
    }

    try {
      await tokenRefreshManager.refresh(refreshAccessToken);

      const newToken = getAccessToken();
      if (!newToken) {
        throw new ConnectError("Token refresh succeeded but no token available", Code.Internal);
      }

      req.header.set("Authorization", `Bearer ${newToken}`);
      req.header.set(RETRY_HEADER, RETRY_HEADER_VALUE);
      return await next(req);
    } catch (refreshError) {
      redirectOnAuthFailure();
      throw refreshError;
    }
  }
};

// ============================================================================
// Transport & Service Clients
// ============================================================================

// Create timeout-enabled fetch for streaming requests
const timeoutFetch = createTimeoutFetch(fetchWithCredentials, DEFAULT_STREAM_TIMEOUT_MS);

// Wrap timeout fetch for streaming requests
const streamingFetch: typeof fetch = timeoutFetch;

const transport = createConnectTransport({
  baseUrl: window.location.origin,
  useBinaryFormat: false,
  fetch: streamingFetch,
  interceptors: [timeoutInterceptor, authInterceptor],
});

// Core service clients
export const instanceServiceClient = createClient(InstanceService, transport);
export const authServiceClient = createClient(AuthService, transport);
export const userServiceClient = createClient(UserService, transport);

// Content service clients
export const memoServiceClient = createClient(MemoService, transport);
export const attachmentServiceClient = createClient(AttachmentService, transport);
export const shortcutServiceClient = createClient(ShortcutService, transport);
export const activityServiceClient = createClient(ActivityService, transport);

// Configuration service clients
export const identityProviderServiceClient = createClient(IdentityProviderService, transport);

// AI service clients
export const aiServiceClient = createClient(AIService, transport);
export const scheduleServiceClient = createClient(ScheduleService, transport);
