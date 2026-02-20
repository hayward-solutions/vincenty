"use client";

import { useState } from "react";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";
import {
  useMapConfigs,
  useCreateMapConfig,
  useDeleteMapConfig,
  useUpdateMapConfig,
} from "@/lib/hooks/use-map-settings";
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
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { MapConfigResponse } from "@/types/api";

export default function MapConfigsPage() {
  const { configs, isLoading, refetch } = useMapConfigs();
  const [createOpen, setCreateOpen] = useState(false);
  const [editConfig, setEditConfig] = useState<MapConfigResponse | null>(null);

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Map Configs</h1>
        <Button onClick={() => setCreateOpen(true)}>Create Config</Button>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
        </div>
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Source Type</TableHead>
                <TableHead>Tile URL</TableHead>
                <TableHead>Zoom</TableHead>
                <TableHead>Default</TableHead>
                <TableHead className="w-12" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {configs.map((config) => (
                <ConfigRow
                  key={config.id}
                  config={config}
                  onEdit={() => setEditConfig(config)}
                  onDeleted={refetch}
                />
              ))}
              {configs.length === 0 && (
                <TableRow>
                  <TableCell
                    colSpan={6}
                    className="text-center text-muted-foreground py-8"
                  >
                    No map configs. The server default tile URL will be used.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      )}

      <CreateConfigDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onCreated={refetch}
      />

      {editConfig && (
        <EditConfigDialog
          config={editConfig}
          open={!!editConfig}
          onOpenChange={(open) => !open && setEditConfig(null)}
          onUpdated={refetch}
        />
      )}
    </div>
  );
}

function ConfigRow({
  config,
  onEdit,
  onDeleted,
}: {
  config: MapConfigResponse;
  onEdit: () => void;
  onDeleted: () => void;
}) {
  const { deleteMapConfig } = useDeleteMapConfig();

  async function handleDelete() {
    if (!confirm(`Delete map config "${config.name}"?`)) return;
    try {
      await deleteMapConfig(config.id);
      toast.success(`Map config "${config.name}" deleted`);
      onDeleted();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to delete map config"
      );
    }
  }

  return (
    <TableRow>
      <TableCell className="font-medium">{config.name}</TableCell>
      <TableCell>
        <Badge variant="secondary">{config.source_type}</Badge>
      </TableCell>
      <TableCell className="max-w-xs truncate text-sm text-muted-foreground">
        {config.tile_url || "(style JSON)"}
      </TableCell>
      <TableCell className="text-sm">
        {config.min_zoom}-{config.max_zoom}
      </TableCell>
      <TableCell>
        {config.is_default && (
          <Badge variant="default">Default</Badge>
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

function CreateConfigDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: () => void;
}) {
  const { createMapConfig, isLoading } = useCreateMapConfig();
  const [form, setForm] = useState({
    name: "",
    source_type: "remote",
    tile_url: "",
    min_zoom: 0,
    max_zoom: 18,
    is_default: false,
  });

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createMapConfig({
        name: form.name,
        source_type: form.source_type,
        tile_url: form.tile_url,
        min_zoom: form.min_zoom,
        max_zoom: form.max_zoom,
        is_default: form.is_default,
      });
      toast.success(`Map config "${form.name}" created`);
      setForm({
        name: "",
        source_type: "remote",
        tile_url: "",
        min_zoom: 0,
        max_zoom: 18,
        is_default: false,
      });
      onOpenChange(false);
      onCreated();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to create map config"
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Map Config</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="cmc-name">Name</Label>
            <Input
              id="cmc-name"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="e.g. OpenStreetMap"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="cmc-type">Source Type</Label>
            <select
              id="cmc-type"
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
              value={form.source_type}
              onChange={(e) =>
                setForm({ ...form, source_type: e.target.value })
              }
            >
              <option value="remote">Remote</option>
              <option value="local">Local (Minio)</option>
              <option value="style">Style JSON</option>
            </select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="cmc-url">Tile URL</Label>
            <Input
              id="cmc-url"
              value={form.tile_url}
              onChange={(e) => setForm({ ...form, tile_url: e.target.value })}
              placeholder="https://tile.openstreetmap.org/{z}/{x}/{y}.png"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="cmc-min">Min Zoom</Label>
              <Input
                id="cmc-min"
                type="number"
                min={0}
                max={24}
                value={form.min_zoom}
                onChange={(e) =>
                  setForm({ ...form, min_zoom: parseInt(e.target.value) || 0 })
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="cmc-max">Max Zoom</Label>
              <Input
                id="cmc-max"
                type="number"
                min={0}
                max={24}
                value={form.max_zoom}
                onChange={(e) =>
                  setForm({
                    ...form,
                    max_zoom: parseInt(e.target.value) || 18,
                  })
                }
              />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <input
              id="cmc-default"
              type="checkbox"
              checked={form.is_default}
              onChange={(e) =>
                setForm({ ...form, is_default: e.target.checked })
              }
              className="h-4 w-4 rounded border-input"
            />
            <Label htmlFor="cmc-default">Set as default</Label>
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

function EditConfigDialog({
  config,
  open,
  onOpenChange,
  onUpdated,
}: {
  config: MapConfigResponse;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUpdated: () => void;
}) {
  const { updateMapConfig, isLoading } = useUpdateMapConfig();
  const [form, setForm] = useState({
    name: config.name,
    source_type: config.source_type,
    tile_url: config.tile_url,
    min_zoom: config.min_zoom,
    max_zoom: config.max_zoom,
    is_default: config.is_default,
  });

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    try {
      await updateMapConfig(config.id, {
        name: form.name,
        source_type: form.source_type,
        tile_url: form.tile_url,
        min_zoom: form.min_zoom,
        max_zoom: form.max_zoom,
        is_default: form.is_default,
      });
      toast.success(`Map config "${form.name}" updated`);
      onOpenChange(false);
      onUpdated();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to update map config"
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit Map Config</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="emc-name">Name</Label>
            <Input
              id="emc-name"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="emc-type">Source Type</Label>
            <select
              id="emc-type"
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
              value={form.source_type}
              onChange={(e) =>
                setForm({ ...form, source_type: e.target.value })
              }
            >
              <option value="remote">Remote</option>
              <option value="local">Local (Minio)</option>
              <option value="style">Style JSON</option>
            </select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="emc-url">Tile URL</Label>
            <Input
              id="emc-url"
              value={form.tile_url}
              onChange={(e) => setForm({ ...form, tile_url: e.target.value })}
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="emc-min">Min Zoom</Label>
              <Input
                id="emc-min"
                type="number"
                min={0}
                max={24}
                value={form.min_zoom}
                onChange={(e) =>
                  setForm({ ...form, min_zoom: parseInt(e.target.value) || 0 })
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="emc-max">Max Zoom</Label>
              <Input
                id="emc-max"
                type="number"
                min={0}
                max={24}
                value={form.max_zoom}
                onChange={(e) =>
                  setForm({
                    ...form,
                    max_zoom: parseInt(e.target.value) || 18,
                  })
                }
              />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <input
              id="emc-default"
              type="checkbox"
              checked={form.is_default}
              onChange={(e) =>
                setForm({ ...form, is_default: e.target.checked })
              }
              className="h-4 w-4 rounded border-input"
            />
            <Label htmlFor="emc-default">Set as default</Label>
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
