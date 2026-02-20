"use client";

import { useState } from "react";
import Link from "next/link";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";
import {
  useGroups,
  useCreateGroup,
  useDeleteGroup,
  useUpdateGroup,
} from "@/lib/hooks/use-groups";
import { Button } from "@/components/ui/button";
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
import type { Group } from "@/types/api";

export default function GroupsPage() {
  const [page, setPage] = useState(1);
  const { data, isLoading, refetch } = useGroups(page);
  const [createOpen, setCreateOpen] = useState(false);
  const [editGroup, setEditGroup] = useState<Group | null>(null);

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Groups</h1>
        <Button onClick={() => setCreateOpen(true)}>Create Group</Button>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
        </div>
      ) : (
        <>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead>Members</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="w-12" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.data.map((group) => (
                  <GroupRow
                    key={group.id}
                    group={group}
                    onEdit={() => setEditGroup(group)}
                    onDeleted={refetch}
                  />
                ))}
                {data?.data.length === 0 && (
                  <TableRow>
                    <TableCell
                      colSpan={5}
                      className="text-center text-muted-foreground py-8"
                    >
                      No groups found
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>

          {data && data.total > data.page_size && (
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                Showing {(page - 1) * data.page_size + 1}-
                {Math.min(page * data.page_size, data.total)} of {data.total}
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
                  disabled={page * data.page_size >= data.total}
                  onClick={() => setPage(page + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </>
      )}

      <CreateGroupDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onCreated={refetch}
      />

      {editGroup && (
        <EditGroupDialog
          group={editGroup}
          open={!!editGroup}
          onOpenChange={(open) => !open && setEditGroup(null)}
          onUpdated={refetch}
        />
      )}
    </div>
  );
}

function GroupRow({
  group,
  onEdit,
  onDeleted,
}: {
  group: Group;
  onEdit: () => void;
  onDeleted: () => void;
}) {
  const { deleteGroup } = useDeleteGroup();

  async function handleDelete() {
    if (!confirm(`Delete group "${group.name}"? This will remove all members.`))
      return;
    try {
      await deleteGroup(group.id);
      toast.success(`Group "${group.name}" deleted`);
      onDeleted();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to delete group"
      );
    }
  }

  return (
    <TableRow>
      <TableCell className="font-medium">
        <Link
          href={`/admin/groups/${group.id}`}
          className="hover:underline text-primary"
        >
          {group.name}
        </Link>
      </TableCell>
      <TableCell className="max-w-xs truncate">
        {group.description || "-"}
      </TableCell>
      <TableCell>{group.member_count}</TableCell>
      <TableCell className="text-sm text-muted-foreground">
        {new Date(group.created_at).toLocaleDateString()}
      </TableCell>
      <TableCell>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm">
              ...
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem asChild>
              <Link href={`/admin/groups/${group.id}`}>Manage Members</Link>
            </DropdownMenuItem>
            <DropdownMenuItem onClick={onEdit}>Edit</DropdownMenuItem>
            <DropdownMenuItem
              onClick={handleDelete}
              className="text-destructive"
            >
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </TableCell>
    </TableRow>
  );
}

function CreateGroupDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: () => void;
}) {
  const { createGroup, isLoading } = useCreateGroup();
  const [form, setForm] = useState({ name: "", description: "" });

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createGroup(form);
      toast.success(`Group "${form.name}" created`);
      setForm({ name: "", description: "" });
      onOpenChange(false);
      onCreated();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to create group"
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Group</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="cg-name">Name</Label>
            <Input
              id="cg-name"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="cg-desc">Description</Label>
            <Input
              id="cg-desc"
              value={form.description}
              onChange={(e) =>
                setForm({ ...form, description: e.target.value })
              }
            />
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
              {isLoading ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function EditGroupDialog({
  group,
  open,
  onOpenChange,
  onUpdated,
}: {
  group: Group;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUpdated: () => void;
}) {
  const { updateGroup, isLoading } = useUpdateGroup();
  const [form, setForm] = useState({
    name: group.name,
    description: group.description,
  });

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    try {
      await updateGroup(group.id, {
        name: form.name,
        description: form.description,
      });
      toast.success(`Group "${form.name}" updated`);
      onOpenChange(false);
      onUpdated();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to update group"
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit Group</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="eg-name">Name</Label>
            <Input
              id="eg-name"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="eg-desc">Description</Label>
            <Input
              id="eg-desc"
              value={form.description}
              onChange={(e) =>
                setForm({ ...form, description: e.target.value })
              }
            />
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
