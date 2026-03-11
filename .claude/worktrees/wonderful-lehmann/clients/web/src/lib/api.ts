const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

interface RequestOptions extends RequestInit {
  params?: Record<string, string>;
}

class ApiClient {
  private baseUrl: string;
  private refreshPromise: Promise<boolean> | null = null;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  private async request<T>(
    path: string,
    options: RequestOptions = {},
    retry = true
  ): Promise<T> {
    const { params, ...init } = options;

    let url = `${this.baseUrl}${path}`;
    if (params) {
      const searchParams = new URLSearchParams(params);
      url += `?${searchParams.toString()}`;
    }

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      ...(init.headers as Record<string, string>),
    };

    if (typeof window !== "undefined") {
      const token = localStorage.getItem("access_token");
      if (token) {
        headers["Authorization"] = `Bearer ${token}`;
      }
    }

    const response = await fetch(url, { ...init, headers, credentials: "include" });

    // Auto-refresh on 401 (once)
    if (response.status === 401 && retry && typeof window !== "undefined") {
      const refreshed = await this.tryRefresh();
      if (refreshed) {
        return this.request<T>(path, options, false);
      }
    }

    if (!response.ok) {
      const body = await response.json().catch(() => ({
        error: { message: response.statusText },
      }));
      const message = body?.error?.message || response.statusText;
      throw new ApiError(response.status, message);
    }

    if (response.status === 204) {
      return undefined as T;
    }

    return response.json();
  }

  private async tryRefresh(): Promise<boolean> {
    // Deduplicate concurrent refresh attempts
    if (this.refreshPromise) return this.refreshPromise;

    this.refreshPromise = (async () => {
      const refreshToken = localStorage.getItem("refresh_token");
      if (!refreshToken) return false;

      try {
        const response = await fetch(`${this.baseUrl}/api/v1/auth/refresh`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ refresh_token: refreshToken }),
          credentials: "include",
        });

        if (!response.ok) {
          this.clearTokens();
          return false;
        }

        const data = await response.json();
        this.setTokens(data.access_token, data.refresh_token);
        return true;
      } catch {
        this.clearTokens();
        return false;
      }
    })();

    const result = await this.refreshPromise;
    this.refreshPromise = null;
    return result;
  }

  setTokens(accessToken: string, refreshToken: string) {
    localStorage.setItem("access_token", accessToken);
    localStorage.setItem("refresh_token", refreshToken);
  }

  clearTokens() {
    localStorage.removeItem("access_token");
    localStorage.removeItem("refresh_token");
  }

  get<T>(path: string, options?: RequestOptions): Promise<T> {
    return this.request<T>(path, { ...options, method: "GET" });
  }

  post<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>(path, {
      ...options,
      method: "POST",
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  put<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>(path, {
      ...options,
      method: "PUT",
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  delete<T>(path: string, options?: RequestOptions): Promise<T> {
    return this.request<T>(path, { ...options, method: "DELETE" });
  }

  /** Upload a file via multipart/form-data (browser sets Content-Type + boundary automatically). */
  async upload<T>(
    path: string,
    formData: FormData,
    method: "POST" | "PUT" = "PUT"
  ): Promise<T> {
    let url = `${this.baseUrl}${path}`;

    const headers: Record<string, string> = {};
    // Do NOT set Content-Type — the browser will set it with the correct boundary
    if (typeof window !== "undefined") {
      const token = localStorage.getItem("access_token");
      if (token) {
        headers["Authorization"] = `Bearer ${token}`;
      }
    }

    const response = await fetch(url, { method, headers, body: formData, credentials: "include" });

    if (response.status === 401 && typeof window !== "undefined") {
      const refreshed = await this.tryRefresh();
      if (refreshed) {
        const retryHeaders: Record<string, string> = {};
        const token = localStorage.getItem("access_token");
        if (token) retryHeaders["Authorization"] = `Bearer ${token}`;
        const retryResponse = await fetch(url, {
          method,
          headers: retryHeaders,
          body: formData,
          credentials: "include",
        });
        if (!retryResponse.ok) {
          const body = await retryResponse.json().catch(() => ({
            error: { message: retryResponse.statusText },
          }));
          throw new ApiError(
            retryResponse.status,
            body?.error?.message || retryResponse.statusText
          );
        }
        if (retryResponse.status === 204) return undefined as T;
        return retryResponse.json();
      }
    }

    if (!response.ok) {
      const body = await response.json().catch(() => ({
        error: { message: response.statusText },
      }));
      throw new ApiError(
        response.status,
        body?.error?.message || response.statusText
      );
    }

    if (response.status === 204) return undefined as T;
    return response.json();
  }
}

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = "ApiError";
  }
}

export const api = new ApiClient(API_BASE);
