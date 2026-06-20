// API client — fetch with Bearer token injection, 401 auto-refresh (once), and logout on failure.
// No external JWT library — role extraction is manual.

import { useAuthStore } from "@/stores/authStore";

export const API_BASE_URL: string = "";

// Queue pattern: concurrent 401 requests share a single refresh call
let isRefreshing = false;
let refreshPromise: Promise<boolean> | null = null;

/** Test-only: inject a mock auth store. NOT for production. */
let _testAuthStore: AuthStoreLike | null = null;

interface AuthStoreLike {
  accessToken: string | null;
  isAuthenticated: boolean;
  refresh: () => Promise<boolean>;
  logout: () => void;
}

function getAuthStore(): AuthStoreLike {
  if (_testAuthStore) return _testAuthStore;
  return useAuthStore.getState();
}

/** @internal Test-only: inject a mock auth store */
export function setAuthStoreForTests(store: AuthStoreLike): void {
  _testAuthStore = store;
}

async function refreshOnce(): Promise<boolean> {
  if (!isRefreshing) {
    isRefreshing = true;
    refreshPromise = getAuthStore().refresh();
  }
  const result: boolean = await refreshPromise!;
  isRefreshing = false;
  refreshPromise = null;
  return result;
}

/** Authenticated fetch — injects Bearer token; 401 → refresh once & retry; refresh failure → logout */
export async function apiClient<T>(
  url: string,
  options: RequestInit & { headers?: Record<string, string> } = {}
): Promise<T> {
  const response = await fetchWithAuth(url, options);

  // Not a 401 — handle normally
  if (response.status !== 401) {
    return handleResponse<T>(response);
  }

  // 401 — attempt refresh and retry
  const refreshed = await refreshOnce();

  if (!refreshed) {
    getAuthStore().logout();
    window.location.href = "/login";
    throw new Error("Session expired");
  }

  // Retry original request with new token
  const retryResponse = await fetchWithAuth(url, options);
  return handleResponse<T>(retryResponse);
}

/** Inject Authorization header from authStore */
async function fetchWithAuth(
  url: string,
  options: RequestInit & { headers?: Record<string, string> } = {}
): Promise<Response> {
  const customHeaders: Record<string, string> = options.headers
    ? { ...options.headers as Record<string, string> }
    : {};
  const restOptions = { ...options };
  delete (restOptions as Record<string, unknown>).headers;

  const auth = getAuthStore();

  const headers: Record<string, string> = {
    ...customHeaders,
  };

  if (restOptions.body) {
    headers["Content-Type"] = "application/json";
  }

  if (auth.isAuthenticated && auth.accessToken) {
    headers.Authorization = `Bearer ${auth.accessToken}`;
  }

  const fullUrl = `${API_BASE_URL}${url}`;
  return fetch(fullUrl, {
    ...restOptions,
    headers,
  });
}

/** Parse response — JSON for success, throw for errors */
async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    // Try to parse error body
    let errorMessage = `Request failed with status ${response.status}`;
    try {
      const errorBody = await response.json() as Record<string, string>;
      if (errorBody.message) {
        errorMessage = errorBody.message;
      } else if (errorBody.error) {
        errorMessage = errorBody.error;
      }
    } catch {
      // Non-JSON error response — use status text
      errorMessage = response.statusText || errorMessage;
    }
    throw new Error(errorMessage);
  }

  // 204 No Content
  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}