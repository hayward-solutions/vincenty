"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import type {
  AddGroupMemberRequest,
  CreateGroupRequest,
  Group,
  GroupMember,
  ListResponse,
  UpdateGroupMemberRequest,
  UpdateGroupRequest,
} from "@/types/api";

export function useGroups(page = 1, pageSize = 20) {
  const [data, setData] = useState<ListResponse<Group> | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchGroups = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<ListResponse<Group>>("/api/v1/groups", {
        params: { page: String(page), page_size: String(pageSize) },
      });
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch groups");
    } finally {
      setIsLoading(false);
    }
  }, [page, pageSize]);

  useEffect(() => {
    fetchGroups();
  }, [fetchGroups]);

  return { data, isLoading, error, refetch: fetchGroups };
}

export function useGroup(id: string) {
  const [group, setGroup] = useState<Group | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchGroup = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<Group>(`/api/v1/groups/${id}`);
      setGroup(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch group");
    } finally {
      setIsLoading(false);
    }
  }, [id]);

  useEffect(() => {
    fetchGroup();
  }, [fetchGroup]);

  return { group, isLoading, error, refetch: fetchGroup };
}

export function useCreateGroup() {
  const [isLoading, setIsLoading] = useState(false);

  const createGroup = async (req: CreateGroupRequest): Promise<Group> => {
    setIsLoading(true);
    try {
      return await api.post<Group>("/api/v1/groups", req);
    } finally {
      setIsLoading(false);
    }
  };

  return { createGroup, isLoading };
}

export function useUpdateGroup() {
  const [isLoading, setIsLoading] = useState(false);

  const updateGroup = async (
    id: string,
    req: UpdateGroupRequest
  ): Promise<Group> => {
    setIsLoading(true);
    try {
      return await api.put<Group>(`/api/v1/groups/${id}`, req);
    } finally {
      setIsLoading(false);
    }
  };

  return { updateGroup, isLoading };
}

export function useDeleteGroup() {
  const [isLoading, setIsLoading] = useState(false);

  const deleteGroup = async (id: string): Promise<void> => {
    setIsLoading(true);
    try {
      await api.delete(`/api/v1/groups/${id}`);
    } finally {
      setIsLoading(false);
    }
  };

  return { deleteGroup, isLoading };
}

// --------------------------------------------------------------------------
// Group Members
// --------------------------------------------------------------------------

export function useGroupMembers(groupId: string) {
  const [members, setMembers] = useState<GroupMember[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchMembers = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<GroupMember[]>(
        `/api/v1/groups/${groupId}/members`
      );
      setMembers(result);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch group members"
      );
    } finally {
      setIsLoading(false);
    }
  }, [groupId]);

  useEffect(() => {
    fetchMembers();
  }, [fetchMembers]);

  return { members, isLoading, error, refetch: fetchMembers };
}

export function useAddGroupMember() {
  const [isLoading, setIsLoading] = useState(false);

  const addMember = async (
    groupId: string,
    req: AddGroupMemberRequest
  ): Promise<GroupMember> => {
    setIsLoading(true);
    try {
      return await api.post<GroupMember>(
        `/api/v1/groups/${groupId}/members`,
        req
      );
    } finally {
      setIsLoading(false);
    }
  };

  return { addMember, isLoading };
}

export function useUpdateGroupMember() {
  const [isLoading, setIsLoading] = useState(false);

  const updateMember = async (
    groupId: string,
    userId: string,
    req: UpdateGroupMemberRequest
  ): Promise<GroupMember> => {
    setIsLoading(true);
    try {
      return await api.put<GroupMember>(
        `/api/v1/groups/${groupId}/members/${userId}`,
        req
      );
    } finally {
      setIsLoading(false);
    }
  };

  return { updateMember, isLoading };
}

export function useRemoveGroupMember() {
  const [isLoading, setIsLoading] = useState(false);

  const removeMember = async (
    groupId: string,
    userId: string
  ): Promise<void> => {
    setIsLoading(true);
    try {
      await api.delete(`/api/v1/groups/${groupId}/members/${userId}`);
    } finally {
      setIsLoading(false);
    }
  };

  return { removeMember, isLoading };
}
