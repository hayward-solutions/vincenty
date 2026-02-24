"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";
import { api } from "@/lib/api";
import {
  useStreamKeys,
  useCreateStreamKey,
  useUpdateStreamKey,
  useDeleteStreamKey,
} from "@/lib/hooks/use-stream-keys";
import type { Group, StreamKeyResponse } from "@/types/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
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
  DialogDescription,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Check, Copy, Key } from "lucide-react";

export default function StreamKeysPage() {
  const { keys, isLoading, refetch } = useStreamKeys();
  const [createOpen, setCreateOpen] = useState(false);
  const [createdKey, setCreatedKey] = useState<string | null>(null);

  return (
    <div className="p-4 md:p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Stream Keys</h1>
        <Button onClick={() => setCreateOpen(true)}>
          <Key className="h-4 w-4 mr-1" />
          Create Stream Key
        </Button>
      </div>

      <p className="text-sm text-muted-foreground">
        Stream keys allow hardware devices (RTSP/RTMP cameras) to publish live
        video without a user account. Streams are automatically shared with the
        configured groups.
      </p>

      {isLoading ? (
        <div className="space-y-2">
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
        </div>
      ) : (
        <div className="rounded-md border overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Label</TableHead>
                <TableHead>Groups</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
                <TableHead className="w-12" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {keys.map((key) => (
                <StreamKeyRow
                  key={key.id}
                  streamKey={key}
                  onUpdated={refetch}
                  onDeleted={refetch}
                />
              ))}
              {keys.length === 0 && (
                <TableRow>
                  <TableCell
                    colSpan={5}
                    className="text-center text-muted-foreground py-8"
                  >
                    No stream keys configured
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      )}

      <CreateStreamKeyDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onCreated={(key) => {
          setCreatedKey(key);
          refetch();
        }}
      />

      {createdKey && (
        <KeyRevealDialog
          plainKey={createdKey}
          onClose={() => setCreatedKey(null)}
        />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Stream Key Row
// ---------------------------------------------------------------------------

function StreamKeyRow({
  streamKey,
  onUpdated,
  onDeleted,
}: {
  streamKey: StreamKeyResponse;
  onUpdated: () => void;
  onDeleted: () => void;
}) {
  const { updateStreamKey } = useUpdateStreamKey();
  const { deleteStreamKey } = useDeleteStreamKey();
  const [groups, setGroups] = useState<Map<string, string>>(new Map());

  // Fetch group names for display
  useEffect(() => {
    if (streamKey.group_ids.length === 0) return;
    // Fetch all groups to map IDs to names
    api
      .get<{ data: Group[] }>("/api/v1/groups", {
        params: { page_size: "100" },
      })
      .then((result) => {
        const map = new Map<string, string>();
        for (const g of result.data) {
          map.set(g.id, g.name);
        }
        setGroups(map);
      })
      .catch(() => {});
  }, [streamKey.group_ids]);

  async function handleToggleActive() {
    try {
      await updateStreamKey(streamKey.id, {
        is_active: !streamKey.is_active,
      });
      toast.success(
        streamKey.is_active ? "Stream key deactivated" : "Stream key activated"
      );
      onUpdated();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to update stream key"
      );
    }
  }

  async function handleDelete() {
    if (
      !confirm(
        `Delete stream key "${streamKey.label}"? Devices using this key will no longer be able to stream.`
      )
    )
      return;
    try {
      await deleteStreamKey(streamKey.id);
      toast.success(`Stream key "${streamKey.label}" deleted`);
      onDeleted();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to delete stream key"
      );
    }
  }

  return (
    <TableRow>
      <TableCell className="font-medium">{streamKey.label}</TableCell>
      <TableCell>
        <div className="flex flex-wrap gap-1">
          {streamKey.group_ids.length === 0 ? (
            <span className="text-muted-foreground text-sm">None</span>
          ) : (
            streamKey.group_ids.map((gid) => (
              <Badge key={gid} variant="secondary" className="text-xs">
                {groups.get(gid) || gid.slice(0, 8)}
              </Badge>
            ))
          )}
        </div>
      </TableCell>
      <TableCell>
        <Badge variant={streamKey.is_active ? "default" : "secondary"}>
          {streamKey.is_active ? "Active" : "Inactive"}
        </Badge>
      </TableCell>
      <TableCell className="text-sm text-muted-foreground">
        {new Date(streamKey.created_at).toLocaleDateString()}
      </TableCell>
      <TableCell>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm">
              ...
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={handleToggleActive}>
              {streamKey.is_active ? "Deactivate" : "Activate"}
            </DropdownMenuItem>
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

// ---------------------------------------------------------------------------
// Create Stream Key Dialog
// ---------------------------------------------------------------------------

function CreateStreamKeyDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (key: string) => void;
}) {
  const { createStreamKey, isLoading } = useCreateStreamKey();
  const [label, setLabel] = useState("");
  const [groups, setGroups] = useState<Group[]>([]);
  const [selectedGroupIds, setSelectedGroupIds] = useState<Set<string>>(
    new Set()
  );

  // Fetch groups for picker
  useEffect(() => {
    if (!open) return;
    api
      .get<{ data: Group[] }>("/api/v1/groups", {
        params: { page_size: "100" },
      })
      .then((result) => setGroups(result.data ?? []))
      .catch(() => setGroups([]));
  }, [open]);

  const toggleGroup = (groupId: string) => {
    setSelectedGroupIds((prev) => {
      const next = new Set(prev);
      if (next.has(groupId)) {
        next.delete(groupId);
      } else {
        next.add(groupId);
      }
      return next;
    });
  };

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    try {
      const result = await createStreamKey({
        label: label.trim(),
        group_ids: Array.from(selectedGroupIds),
      });
      setLabel("");
      setSelectedGroupIds(new Set());
      onOpenChange(false);
      if (result.key) {
        onCreated(result.key);
      }
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to create stream key"
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Stream Key</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="sk-label">Label</Label>
            <Input
              id="sk-label"
              placeholder="e.g. Front Gate Camera"
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              required
            />
          </div>
          <div className="space-y-2">
            <Label>Default Groups</Label>
            <p className="text-xs text-muted-foreground">
              Streams from this key will be automatically shared with these
              groups.
            </p>
            {groups.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No groups available
              </p>
            ) : (
              <ScrollArea className="max-h-40">
                <div className="flex flex-wrap gap-1.5">
                  {groups.map((group) => {
                    const selected = selectedGroupIds.has(group.id);
                    return (
                      <Badge
                        key={group.id}
                        variant={selected ? "default" : "outline"}
                        className="cursor-pointer select-none"
                        onClick={() => toggleGroup(group.id)}
                      >
                        {selected && <Check className="h-3 w-3 mr-1" />}
                        {group.name}
                      </Badge>
                    );
                  })}
                </div>
              </ScrollArea>
            )}
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading || !label.trim()}>
              {isLoading ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Key Reveal Dialog (shown once after creation)
// ---------------------------------------------------------------------------

function KeyRevealDialog({
  plainKey,
  onClose,
}: {
  plainKey: string;
  onClose: () => void;
}) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(plainKey);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error("Failed to copy to clipboard");
    }
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Stream Key Created</DialogTitle>
          <DialogDescription>
            Copy this key now. It will not be shown again.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <code className="flex-1 bg-muted px-3 py-2 rounded-md text-sm font-mono break-all select-all">
              {plainKey}
            </code>
            <Button
              variant="outline"
              size="sm"
              onClick={handleCopy}
              className="flex-shrink-0"
            >
              {copied ? (
                <Check className="h-4 w-4" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </Button>
          </div>
          <p className="text-xs text-muted-foreground">
            Use this key as the password when configuring your RTSP/RTMP device.
            The username can be anything.
          </p>
        </div>
        <DialogFooter>
          <Button onClick={onClose}>Done</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
