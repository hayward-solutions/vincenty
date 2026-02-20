"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import type {
  CreateUserRequest,
  ListResponse,
  UpdateUserRequest,
  User,
} from "@/types/api";

export function useUsers(page = 1, pageSize = 20) {
  const [data, setData] = useState<ListResponse<User> | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchUsers = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<ListResponse<User>>("/api/v1/users", {
        params: { page: String(page), page_size: String(pageSize) },
      });
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch users");
    } finally {
      setIsLoading(false);
    }
  }, [page, pageSize]);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  return { data, isLoading, error, refetch: fetchUsers };
}

export function useCreateUser() {
  const [isLoading, setIsLoading] = useState(false);

  const createUser = async (req: CreateUserRequest): Promise<User> => {
    setIsLoading(true);
    try {
      return await api.post<User>("/api/v1/users", req);
    } finally {
      setIsLoading(false);
    }
  };

  return { createUser, isLoading };
}

export function useUpdateUser() {
  const [isLoading, setIsLoading] = useState(false);

  const updateUser = async (
    id: string,
    req: UpdateUserRequest
  ): Promise<User> => {
    setIsLoading(true);
    try {
      return await api.put<User>(`/api/v1/users/${id}`, req);
    } finally {
      setIsLoading(false);
    }
  };

  return { updateUser, isLoading };
}

export function useDeleteUser() {
  const [isLoading, setIsLoading] = useState(false);

  const deleteUser = async (id: string): Promise<void> => {
    setIsLoading(true);
    try {
      await api.delete(`/api/v1/users/${id}`);
    } finally {
      setIsLoading(false);
    }
  };

  return { deleteUser, isLoading };
}
