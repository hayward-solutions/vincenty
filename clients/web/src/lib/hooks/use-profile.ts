"use client";

import { useState } from "react";
import { api } from "@/lib/api";
import type { ChangePasswordRequest, UpdateMeRequest, User } from "@/types/api";

export function useUpdateMe() {
  const [isLoading, setIsLoading] = useState(false);

  const updateMe = async (req: UpdateMeRequest): Promise<User> => {
    setIsLoading(true);
    try {
      return await api.put<User>("/api/v1/users/me", req);
    } finally {
      setIsLoading(false);
    }
  };

  return { updateMe, isLoading };
}

export function useChangePassword() {
  const [isLoading, setIsLoading] = useState(false);

  const changePassword = async (req: ChangePasswordRequest): Promise<void> => {
    setIsLoading(true);
    try {
      await api.put<{ message: string }>("/api/v1/users/me/password", req);
    } finally {
      setIsLoading(false);
    }
  };

  return { changePassword, isLoading };
}

export function useUploadAvatar() {
  const [isLoading, setIsLoading] = useState(false);

  const uploadAvatar = async (file: File): Promise<User> => {
    setIsLoading(true);
    try {
      const formData = new FormData();
      formData.append("avatar", file);
      return await api.upload<User>("/api/v1/users/me/avatar", formData, "PUT");
    } finally {
      setIsLoading(false);
    }
  };

  return { uploadAvatar, isLoading };
}

export function useDeleteAvatar() {
  const [isLoading, setIsLoading] = useState(false);

  const deleteAvatar = async (): Promise<User> => {
    setIsLoading(true);
    try {
      return await api.delete<User>("/api/v1/users/me/avatar");
    } finally {
      setIsLoading(false);
    }
  };

  return { deleteAvatar, isLoading };
}
