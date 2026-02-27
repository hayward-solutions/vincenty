"use client";

import { useCallback, useState } from "react";
import { api } from "@/lib/api";
import type { ApiToken, CreateApiTokenRequest, CreateApiTokenResponse } from "@/types/api";

/**
 * Fetch the authenticated user's API tokens.
 * The endpoint returns a plain ApiToken[] (not paginated).
 */
export function useApiTokens() {
  const [tokens, setTokens] = useState<ApiToken[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetch = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<ApiToken[]>("/api/v1/users/me/api-tokens");
      setTokens(result ?? []);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch API tokens"
      );
    } finally {
      setIsLoading(false);
    }
  }, []);

  return { tokens, isLoading, error, fetch };
}

/** Create a new API token via POST /api/v1/users/me/api-tokens. */
export function useCreateApiToken() {
  const [isLoading, setIsLoading] = useState(false);

  const createToken = async (
    request: CreateApiTokenRequest
  ): Promise<CreateApiTokenResponse> => {
    setIsLoading(true);
    try {
      return await api.post<CreateApiTokenResponse>(
        "/api/v1/users/me/api-tokens",
        request
      );
    } finally {
      setIsLoading(false);
    }
  };

  return { createToken, isLoading };
}

/** Delete an API token by ID via DELETE /api/v1/users/me/api-tokens/{id}. */
export function useDeleteApiToken() {
  const [isLoading, setIsLoading] = useState(false);

  const deleteToken = async (id: string): Promise<void> => {
    setIsLoading(true);
    try {
      await api.delete(`/api/v1/users/me/api-tokens/${id}`);
    } finally {
      setIsLoading(false);
    }
  };

  return { deleteToken, isLoading };
}
