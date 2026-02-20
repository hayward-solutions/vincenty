"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import type { Group, GroupMember, ListResponse, User } from "@/types/api";
import { Loader2, User as UserIcon } from "lucide-react";

interface DmUser {
  id: string;
  username: string;
  displayName: string;
}

interface NewDmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSelect: (userId: string, displayName: string) => void;
}

export function NewDmDialog({
  open,
  onOpenChange,
  onSelect,
}: NewDmDialogProps) {
  const { user, isAdmin } = useAuth();
  const [users, setUsers] = useState<DmUser[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [search, setSearch] = useState("");

  const fetchUsers = useCallback(async () => {
    if (!user) return;
    setIsLoading(true);
    try {
      if (isAdmin) {
        // Admin: fetch all users
        const result = await api.get<ListResponse<User>>("/api/v1/users", {
          params: { page: "1", page_size: "200" },
        });
        setUsers(
          result.data
            .filter((u) => u.id !== user.id && u.is_active)
            .map((u) => ({
              id: u.id,
              username: u.username,
              displayName: u.display_name || u.username,
            }))
        );
      } else {
        // Regular user: fetch members from all groups, deduplicate
        const groups = await api.get<Group[]>("/api/v1/users/me/groups");

        const memberMap = new Map<string, DmUser>();
        await Promise.all(
          groups.map(async (g) => {
            try {
              const members = await api.get<GroupMember[]>(
                `/api/v1/groups/${g.id}/members`
              );
              for (const m of members) {
                if (m.user_id !== user.id && !memberMap.has(m.user_id)) {
                  memberMap.set(m.user_id, {
                    id: m.user_id,
                    username: m.username,
                    displayName: m.display_name || m.username,
                  });
                }
              }
            } catch {
              // Skip groups we can't read members for
            }
          })
        );

        setUsers(
          Array.from(memberMap.values()).sort((a, b) =>
            a.username.localeCompare(b.username)
          )
        );
      }
    } catch (err) {
      console.error("Failed to fetch users for DM:", err);
    } finally {
      setIsLoading(false);
    }
  }, [user, isAdmin]);

  // Fetch when dialog opens
  useEffect(() => {
    if (open) {
      setSearch("");
      fetchUsers();
    }
  }, [open, fetchUsers]);

  const filtered = useMemo(() => {
    if (!search.trim()) return users;
    const q = search.toLowerCase();
    return users.filter(
      (u) =>
        u.username.toLowerCase().includes(q) ||
        u.displayName.toLowerCase().includes(q)
    );
  }, [users, search]);

  const handleSelect = (u: DmUser) => {
    onSelect(u.id, u.displayName);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>New Direct Message</DialogTitle>
        </DialogHeader>

        <Input
          placeholder="Search users..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          autoFocus
        />

        <ScrollArea className="h-64">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : filtered.length === 0 ? (
            <div className="flex items-center justify-center py-8 text-muted-foreground text-sm">
              {users.length === 0
                ? "No users available to message"
                : "No matching users"}
            </div>
          ) : (
            <div className="flex flex-col gap-0.5">
              {filtered.map((u) => (
                <button
                  type="button"
                  key={u.id}
                  onClick={() => handleSelect(u)}
                  className="flex items-center gap-2.5 rounded-md px-3 py-2 text-sm text-left transition-colors hover:bg-accent hover:text-accent-foreground"
                >
                  <UserIcon className="h-4 w-4 shrink-0 text-muted-foreground" />
                  <div className="min-w-0">
                    <div className="font-medium truncate">
                      {u.displayName}
                    </div>
                    {u.displayName !== u.username && (
                      <div className="text-xs text-muted-foreground truncate">
                        @{u.username}
                      </div>
                    )}
                  </div>
                </button>
              ))}
            </div>
          )}
        </ScrollArea>
      </DialogContent>
    </Dialog>
  );
}
