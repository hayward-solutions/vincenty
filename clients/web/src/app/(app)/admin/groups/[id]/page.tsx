"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";
import { useGroup } from "@/lib/hooks/use-groups";
import {
  useGroupMembers,
  useAddGroupMember,
  useUpdateGroupMember,
  useRemoveGroupMember,
} from "@/lib/hooks/use-groups";
import { useUsers } from "@/lib/hooks/use-users";
import { useGroupAuditLogs } from "@/lib/hooks/use-audit-logs";
import { AuditLogTable } from "@/components/audit/audit-log-table";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { GroupMember } from "@/types/api";

export default function GroupDetailPage() {
  const params = useParams();
  const router = useRouter();
  const groupId = params.id as string;

  const { group, isLoading: groupLoading } = useGroup(groupId);
  const { members, isLoading: membersLoading, refetch } = useGroupMembers(groupId);
  const [addOpen, setAddOpen] = useState(false);
  const [editMember, setEditMember] = useState<GroupMember | null>(null);

  if (groupLoading) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
      </div>
    );
  }

  if (!group) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">Group not found</p>
        <Button variant="outline" className="mt-4" asChild>
          <Link href="/admin/groups">Back to Groups</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="sm" onClick={() => router.push("/admin/groups")}>
          &larr; Groups
        </Button>
      </div>

      <div className="space-y-1">
        <h1 className="text-2xl font-semibold">{group.name}</h1>
        {group.description && (
          <p className="text-muted-foreground">{group.description}</p>
        )}
      </div>

      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-medium">
            Members ({members.length})
          </h2>
          <Button onClick={() => setAddOpen(true)}>Add Member</Button>
        </div>

        {membersLoading ? (
          <div className="space-y-2">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : (
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>User</TableHead>
                  <TableHead>Read</TableHead>
                  <TableHead>Write</TableHead>
                  <TableHead>Group Admin</TableHead>
                  <TableHead className="w-12" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {members.map((member) => (
                  <MemberRow
                    key={member.id}
                    member={member}
                    groupId={groupId}
                    onEdit={() => setEditMember(member)}
                    onRemoved={refetch}
                  />
                ))}
                {members.length === 0 && (
                  <TableRow>
                    <TableCell
                      colSpan={5}
                      className="text-center text-muted-foreground py-8"
                    >
                      No members yet. Add members to this group.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        )}
      </div>

      <GroupAuditSection groupId={groupId} />

      <AddMemberDialog
        groupId={groupId}
        existingMemberIds={members.map((m) => m.user_id)}
        open={addOpen}
        onOpenChange={setAddOpen}
        onAdded={refetch}
      />

      {editMember && (
        <EditMemberDialog
          groupId={groupId}
          member={editMember}
          open={!!editMember}
          onOpenChange={(open) => !open && setEditMember(null)}
          onUpdated={refetch}
        />
      )}
    </div>
  );
}

function GroupAuditSection({ groupId }: { groupId: string }) {
  const { data, total, isLoading, error, fetch } = useGroupAuditLogs();
  const [page, setPage] = useState(1);
  const pageSize = 10;

  const loadPage = useCallback(
    (p: number) => {
      fetch(groupId, { page: p, page_size: pageSize });
    },
    [fetch, groupId]
  );

  useEffect(() => {
    loadPage(page);
  }, [page, loadPage]);

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-medium">Audit Log</h2>

      {error && <p className="text-sm text-destructive">{error}</p>}

      {isLoading ? (
        <div className="space-y-2">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      ) : (
        <>
          <AuditLogTable logs={data} showUser />

          {total > pageSize && (
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                Showing {(page - 1) * pageSize + 1}-
                {Math.min(page * pageSize, total)} of {total}
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage(page - 1)}
                >
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page * pageSize >= total}
                  onClick={() => setPage(page + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}

function MemberRow({
  member,
  groupId,
  onEdit,
  onRemoved,
}: {
  member: GroupMember;
  groupId: string;
  onEdit: () => void;
  onRemoved: () => void;
}) {
  const { removeMember } = useRemoveGroupMember();

  async function handleRemove() {
    if (
      !confirm(
        `Remove "${member.display_name || member.username}" from this group?`
      )
    )
      return;
    try {
      await removeMember(groupId, member.user_id);
      toast.success(`Member "${member.username}" removed`);
      onRemoved();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to remove member"
      );
    }
  }

  return (
    <TableRow>
      <TableCell>
        <div>
          <span className="font-medium">{member.username}</span>
          {member.display_name && (
            <span className="text-muted-foreground ml-2">
              ({member.display_name})
            </span>
          )}
        </div>
      </TableCell>
      <TableCell>
        {member.can_read ? (
          <Badge variant="outline" className="border-green-500 text-green-500">
            Yes
          </Badge>
        ) : (
          <Badge variant="outline" className="border-red-500 text-red-500">
            No
          </Badge>
        )}
      </TableCell>
      <TableCell>
        {member.can_write ? (
          <Badge variant="outline" className="border-green-500 text-green-500">
            Yes
          </Badge>
        ) : (
          <Badge variant="outline" className="border-red-500 text-red-500">
            No
          </Badge>
        )}
      </TableCell>
      <TableCell>
        {member.is_group_admin ? (
          <Badge variant="default">Admin</Badge>
        ) : (
          <Badge variant="secondary">Member</Badge>
        )}
      </TableCell>
      <TableCell>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm">
              ...
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={onEdit}>
              Edit Permissions
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={handleRemove}
              className="text-destructive"
            >
              Remove
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </TableCell>
    </TableRow>
  );
}

function AddMemberDialog({
  groupId,
  existingMemberIds,
  open,
  onOpenChange,
  onAdded,
}: {
  groupId: string;
  existingMemberIds: string[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAdded: () => void;
}) {
  const { addMember, isLoading } = useAddGroupMember();
  const { data: usersData } = useUsers(1, 100);
  const [selectedUserId, setSelectedUserId] = useState("");
  const [canRead, setCanRead] = useState(true);
  const [canWrite, setCanWrite] = useState(false);
  const [isGroupAdmin, setIsGroupAdmin] = useState(false);

  // Filter out users who are already members
  const availableUsers =
    usersData?.data.filter(
      (u) => u.is_active && !existingMemberIds.includes(u.id)
    ) ?? [];

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!selectedUserId) {
      toast.error("Please select a user");
      return;
    }
    try {
      await addMember(groupId, {
        user_id: selectedUserId,
        can_read: canRead,
        can_write: canWrite,
        is_group_admin: isGroupAdmin,
      });
      toast.success("Member added");
      setSelectedUserId("");
      setCanRead(true);
      setCanWrite(false);
      setIsGroupAdmin(false);
      onOpenChange(false);
      onAdded();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to add member"
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Member</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="am-user">User</Label>
            <select
              id="am-user"
              value={selectedUserId}
              onChange={(e) => setSelectedUserId(e.target.value)}
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              required
            >
              <option value="">Select a user...</option>
              {availableUsers.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.username}
                  {u.display_name ? ` (${u.display_name})` : ""}
                </option>
              ))}
            </select>
            {availableUsers.length === 0 && (
              <p className="text-sm text-muted-foreground">
                No available users to add.
              </p>
            )}
          </div>
          <div className="space-y-3">
            <Label>Permissions</Label>
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="am-read"
                  checked={canRead}
                  onChange={(e) => setCanRead(e.target.checked)}
                  className="h-4 w-4"
                />
                <Label htmlFor="am-read">Read</Label>
              </div>
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="am-write"
                  checked={canWrite}
                  onChange={(e) => setCanWrite(e.target.checked)}
                  className="h-4 w-4"
                />
                <Label htmlFor="am-write">Write</Label>
              </div>
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="am-admin"
                  checked={isGroupAdmin}
                  onChange={(e) => setIsGroupAdmin(e.target.checked)}
                  className="h-4 w-4"
                />
                <Label htmlFor="am-admin">Group Admin</Label>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={isLoading || !selectedUserId}
            >
              {isLoading ? "Adding..." : "Add Member"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function EditMemberDialog({
  groupId,
  member,
  open,
  onOpenChange,
  onUpdated,
}: {
  groupId: string;
  member: GroupMember;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUpdated: () => void;
}) {
  const { updateMember, isLoading } = useUpdateGroupMember();
  const [canRead, setCanRead] = useState(member.can_read);
  const [canWrite, setCanWrite] = useState(member.can_write);
  const [isGroupAdmin, setIsGroupAdmin] = useState(member.is_group_admin);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    try {
      await updateMember(groupId, member.user_id, {
        can_read: canRead,
        can_write: canWrite,
        is_group_admin: isGroupAdmin,
      });
      toast.success(`Permissions updated for "${member.username}"`);
      onOpenChange(false);
      onUpdated();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to update member"
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            Edit Permissions - {member.display_name || member.username}
          </DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="em-read"
                checked={canRead}
                onChange={(e) => setCanRead(e.target.checked)}
                className="h-4 w-4"
              />
              <Label htmlFor="em-read">Can Read</Label>
            </div>
            <p className="text-sm text-muted-foreground ml-6">
              View messages and locations in this group
            </p>

            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="em-write"
                checked={canWrite}
                onChange={(e) => setCanWrite(e.target.checked)}
                className="h-4 w-4"
              />
              <Label htmlFor="em-write">Can Write</Label>
            </div>
            <p className="text-sm text-muted-foreground ml-6">
              Send messages and share location in this group
            </p>

            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="em-admin"
                checked={isGroupAdmin}
                onChange={(e) => setIsGroupAdmin(e.target.checked)}
                className="h-4 w-4"
              />
              <Label htmlFor="em-admin">Group Admin</Label>
            </div>
            <p className="text-sm text-muted-foreground ml-6">
              Can add/remove members and manage permissions within this group
            </p>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? "Saving..." : "Save"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
