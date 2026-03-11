"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Eye, EyeOff } from "lucide-react";
import { ApiError } from "@/lib/api";
import {
  useMapConfigs,
  useCreateMapConfig,
  useDeleteMapConfig,
  useUpdateMapConfig,
  useTerrainConfigs,
  useCreateTerrainConfig,
  useDeleteTerrainConfig,
  useUpdateTerrainConfig,
} from "@/lib/hooks/use-map-settings";
import { useServerSettings } from "@/lib/hooks/use-mfa";
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
import type { MapConfigResponse, TerrainConfigResponse } from "@/types/api";

// ===========================================================================
// Page
// ===========================================================================

export default function MapSettingsPage() {
  const { configs, isLoading, refetch } = useMapConfigs();
  const [createOpen, setCreateOpen] = useState(false);
  const [editConfig, setEditConfig] = useState<MapConfigResponse | null>(null);

  const {
    configs: terrainConfigs,
    isLoading: terrainLoading,
    refetch: refetchTerrain,
  } = useTerrainConfigs();
  const [createTerrainOpen, setCreateTerrainOpen] = useState(false);
  const [editTerrainConfig, setEditTerrainConfig] =
    useState<TerrainConfigResponse | null>(null);

  return (
    <div className="p-4 md:p-6 space-y-8">
      {/* --------------------------------------------------------------- */}
      {/* API Keys                                                         */}
      {/* --------------------------------------------------------------- */}
      <ApiKeysSection />

      {/* --------------------------------------------------------------- */}
      {/* Tile Configs                                                     */}
      {/* --------------------------------------------------------------- */}
      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-semibold">Tile Configs</h1>
          <Button onClick={() => setCreateOpen(true)}>Create Config</Button>
        </div>

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
                  <TableHead className="w-[20%]">Name</TableHead>
                  <TableHead className="w-[10%]">Source Type</TableHead>
                  <TableHead>Tile URL</TableHead>
                  <TableHead className="w-[10%]">Zoom</TableHead>
                  <TableHead className="w-[12%]">Status</TableHead>
                  <TableHead className="w-12" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {configs.map((config) => (
                  <TileConfigRow
                    key={config.id}
                    config={config}
                    onEdit={() => setEditConfig(config)}
                    onChanged={refetch}
                  />
                ))}
              </TableBody>
            </Table>
          </div>
        )}

        <CreateTileConfigDialog
          open={createOpen}
          onOpenChange={setCreateOpen}
          onCreated={refetch}
        />

        {editConfig && (
          <EditTileConfigDialog
            config={editConfig}
            open={!!editConfig}
            onOpenChange={(open) => !open && setEditConfig(null)}
            onUpdated={refetch}
          />
        )}
      </section>

      {/* --------------------------------------------------------------- */}
      {/* Terrain Configs                                                  */}
      {/* --------------------------------------------------------------- */}
      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-semibold">Terrain Configs</h2>
          <Button onClick={() => setCreateTerrainOpen(true)}>
            Create Config
          </Button>
        </div>

        {terrainLoading ? (
          <div className="space-y-2">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : (
          <div className="rounded-md border overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[20%]">Name</TableHead>
                  <TableHead className="w-[10%]">Source Type</TableHead>
                  <TableHead>Terrain URL</TableHead>
                  <TableHead className="w-[10%]">Encoding</TableHead>
                  <TableHead className="w-[12%]">Status</TableHead>
                  <TableHead className="w-12" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {terrainConfigs.map((config) => (
                  <TerrainConfigRow
                    key={config.id}
                    config={config}
                    onEdit={() => setEditTerrainConfig(config)}
                    onChanged={refetchTerrain}
                  />
                ))}
              </TableBody>
            </Table>
          </div>
        )}

        <CreateTerrainConfigDialog
          open={createTerrainOpen}
          onOpenChange={setCreateTerrainOpen}
          onCreated={refetchTerrain}
        />

        {editTerrainConfig && (
          <EditTerrainConfigDialog
            config={editTerrainConfig}
            open={!!editTerrainConfig}
            onOpenChange={(open) => !open && setEditTerrainConfig(null)}
            onUpdated={refetchTerrain}
          />
        )}
      </section>
    </div>
  );
}

// ===========================================================================
// Tile config rows
// ===========================================================================

function TileConfigRow({
  config,
  onEdit,
  onChanged,
}: {
  config: MapConfigResponse;
  onEdit: () => void;
  onChanged: () => void;
}) {
  const { deleteMapConfig } = useDeleteMapConfig();
  const { updateMapConfig } = useUpdateMapConfig();

  async function handleDelete() {
    if (!confirm(`Delete tile config "${config.name}"?`)) return;
    try {
      await deleteMapConfig(config.id);
      toast.success(`Tile config "${config.name}" deleted`);
      onChanged();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to delete tile config"
      );
    }
  }

  async function handleSetDefault() {
    try {
      await updateMapConfig(config.id, { is_default: true });
      toast.success(`"${config.name}" set as default`);
      onChanged();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to set default"
      );
    }
  }

  async function handleToggleEnabled() {
    try {
      await updateMapConfig(config.id, { is_enabled: !config.is_enabled });
      toast.success(
        `"${config.name}" ${config.is_enabled ? "disabled" : "enabled"}`
      );
      onChanged();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to update config"
      );
    }
  }

  const isDisabled = !config.is_enabled;

  return (
    <TableRow className={isDisabled ? "text-muted-foreground opacity-60" : ""}>
      <TableCell className="font-medium">
        {config.name}
        {config.is_builtin && (
          <Badge variant="outline" className="ml-2">
            Built-in
          </Badge>
        )}
      </TableCell>
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
        <div className="flex items-center gap-1.5">
          {config.is_default && <Badge variant="default">Default</Badge>}
          {isDisabled && <Badge variant="secondary">Disabled</Badge>}
        </div>
      </TableCell>
      <TableCell>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm">
              ...
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {!config.is_default && config.is_enabled && (
              <DropdownMenuItem onClick={handleSetDefault}>
                Set as Default
              </DropdownMenuItem>
            )}
            {!config.is_builtin && (
              <DropdownMenuItem onClick={onEdit}>Edit</DropdownMenuItem>
            )}
            <DropdownMenuItem onClick={handleToggleEnabled}>
              {config.is_enabled ? "Disable" : "Enable"}
            </DropdownMenuItem>
            {!config.is_builtin && (
              <DropdownMenuItem
                onClick={handleDelete}
                className="text-destructive"
              >
                Delete
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      </TableCell>
    </TableRow>
  );
}

// ===========================================================================
// Tile config dialogs
// ===========================================================================

function CreateTileConfigDialog({
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
      toast.success(`Tile config "${form.name}" created`);
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
        err instanceof ApiError ? err.message : "Failed to create tile config"
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Tile Config</DialogTitle>
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
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
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

function EditTileConfigDialog({
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
      toast.success(`Tile config "${form.name}" updated`);
      onOpenChange(false);
      onUpdated();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to update tile config"
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit Tile Config</DialogTitle>
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
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
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

// ===========================================================================
// Terrain config rows
// ===========================================================================

function TerrainConfigRow({
  config,
  onEdit,
  onChanged,
}: {
  config: TerrainConfigResponse;
  onEdit: () => void;
  onChanged: () => void;
}) {
  const { deleteTerrainConfig } = useDeleteTerrainConfig();
  const { updateTerrainConfig } = useUpdateTerrainConfig();

  async function handleDelete() {
    if (!confirm(`Delete terrain config "${config.name}"?`)) return;
    try {
      await deleteTerrainConfig(config.id);
      toast.success(`Terrain config "${config.name}" deleted`);
      onChanged();
    } catch (err) {
      toast.error(
        err instanceof ApiError
          ? err.message
          : "Failed to delete terrain config"
      );
    }
  }

  async function handleSetDefault() {
    try {
      await updateTerrainConfig(config.id, { is_default: true });
      toast.success(`"${config.name}" set as default`);
      onChanged();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to set default"
      );
    }
  }

  async function handleToggleEnabled() {
    try {
      await updateTerrainConfig(config.id, {
        is_enabled: !config.is_enabled,
      });
      toast.success(
        `"${config.name}" ${config.is_enabled ? "disabled" : "enabled"}`
      );
      onChanged();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to update config"
      );
    }
  }

  const isDisabled = !config.is_enabled;

  return (
    <TableRow className={isDisabled ? "text-muted-foreground opacity-60" : ""}>
      <TableCell className="font-medium">
        {config.name}
        {config.is_builtin && (
          <Badge variant="outline" className="ml-2">
            Built-in
          </Badge>
        )}
      </TableCell>
      <TableCell>
        <Badge variant="secondary">{config.source_type}</Badge>
      </TableCell>
      <TableCell className="max-w-xs truncate text-sm text-muted-foreground">
        {config.terrain_url}
      </TableCell>
      <TableCell className="text-sm">{config.terrain_encoding}</TableCell>
      <TableCell>
        <div className="flex items-center gap-1.5">
          {config.is_default && <Badge variant="default">Default</Badge>}
          {isDisabled && <Badge variant="secondary">Disabled</Badge>}
        </div>
      </TableCell>
      <TableCell>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm">
              ...
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {!config.is_default && config.is_enabled && (
              <DropdownMenuItem onClick={handleSetDefault}>
                Set as Default
              </DropdownMenuItem>
            )}
            {!config.is_builtin && (
              <DropdownMenuItem onClick={onEdit}>Edit</DropdownMenuItem>
            )}
            <DropdownMenuItem onClick={handleToggleEnabled}>
              {config.is_enabled ? "Disable" : "Enable"}
            </DropdownMenuItem>
            {!config.is_builtin && (
              <DropdownMenuItem
                onClick={handleDelete}
                className="text-destructive"
              >
                Delete
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      </TableCell>
    </TableRow>
  );
}

// ===========================================================================
// Terrain config dialogs
// ===========================================================================

function CreateTerrainConfigDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: () => void;
}) {
  const { createTerrainConfig, isLoading } = useCreateTerrainConfig();
  const [form, setForm] = useState({
    name: "",
    source_type: "remote",
    terrain_url: "",
    terrain_encoding: "terrarium",
    is_default: false,
  });

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createTerrainConfig({
        name: form.name,
        source_type: form.source_type,
        terrain_url: form.terrain_url,
        terrain_encoding: form.terrain_encoding,
        is_default: form.is_default,
      });
      toast.success(`Terrain config "${form.name}" created`);
      setForm({
        name: "",
        source_type: "remote",
        terrain_url: "",
        terrain_encoding: "terrarium",
        is_default: false,
      });
      onOpenChange(false);
      onCreated();
    } catch (err) {
      toast.error(
        err instanceof ApiError
          ? err.message
          : "Failed to create terrain config"
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Terrain Config</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="ctc-name">Name</Label>
            <Input
              id="ctc-name"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="e.g. AWS Terrarium"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="ctc-type">Source Type</Label>
            <select
              id="ctc-type"
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
              value={form.source_type}
              onChange={(e) =>
                setForm({ ...form, source_type: e.target.value })
              }
            >
              <option value="remote">Remote</option>
              <option value="local">Local (Minio)</option>
            </select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="ctc-url">Terrain URL</Label>
            <Input
              id="ctc-url"
              value={form.terrain_url}
              onChange={(e) =>
                setForm({ ...form, terrain_url: e.target.value })
              }
              placeholder="https://s3.amazonaws.com/elevation-tiles-prod/terrarium/{z}/{x}/{y}.png"
              required
            />
            <p className="text-xs text-muted-foreground">
              DEM tile URL for 3D terrain elevation data.
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="ctc-enc">Encoding</Label>
            <select
              id="ctc-enc"
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
              value={form.terrain_encoding}
              onChange={(e) =>
                setForm({ ...form, terrain_encoding: e.target.value })
              }
            >
              <option value="terrarium">Terrarium (AWS/Mapzen)</option>
              <option value="mapbox">Mapbox (MapTiler)</option>
            </select>
          </div>
          <div className="flex items-center gap-2">
            <input
              id="ctc-default"
              type="checkbox"
              checked={form.is_default}
              onChange={(e) =>
                setForm({ ...form, is_default: e.target.checked })
              }
              className="h-4 w-4 rounded border-input"
            />
            <Label htmlFor="ctc-default">Set as default</Label>
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

function EditTerrainConfigDialog({
  config,
  open,
  onOpenChange,
  onUpdated,
}: {
  config: TerrainConfigResponse;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUpdated: () => void;
}) {
  const { updateTerrainConfig, isLoading } = useUpdateTerrainConfig();
  const [form, setForm] = useState({
    name: config.name,
    source_type: config.source_type || "remote",
    terrain_url: config.terrain_url,
    terrain_encoding: config.terrain_encoding || "terrarium",
    is_default: config.is_default,
  });

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    try {
      await updateTerrainConfig(config.id, {
        name: form.name,
        source_type: form.source_type,
        terrain_url: form.terrain_url,
        terrain_encoding: form.terrain_encoding,
        is_default: form.is_default,
      });
      toast.success(`Terrain config "${form.name}" updated`);
      onOpenChange(false);
      onUpdated();
    } catch (err) {
      toast.error(
        err instanceof ApiError
          ? err.message
          : "Failed to update terrain config"
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit Terrain Config</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="etc-name">Name</Label>
            <Input
              id="etc-name"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="etc-type">Source Type</Label>
            <select
              id="etc-type"
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
              value={form.source_type}
              onChange={(e) =>
                setForm({ ...form, source_type: e.target.value })
              }
            >
              <option value="remote">Remote</option>
              <option value="local">Local (Minio)</option>
            </select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="etc-url">Terrain URL</Label>
            <Input
              id="etc-url"
              value={form.terrain_url}
              onChange={(e) =>
                setForm({ ...form, terrain_url: e.target.value })
              }
              placeholder="https://s3.amazonaws.com/elevation-tiles-prod/terrarium/{z}/{x}/{y}.png"
            />
            <p className="text-xs text-muted-foreground">
              DEM tile URL for 3D terrain elevation data.
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="etc-enc">Encoding</Label>
            <select
              id="etc-enc"
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
              value={form.terrain_encoding}
              onChange={(e) =>
                setForm({ ...form, terrain_encoding: e.target.value })
              }
            >
              <option value="terrarium">Terrarium (AWS/Mapzen)</option>
              <option value="mapbox">Mapbox (MapTiler)</option>
            </select>
          </div>
          <div className="flex items-center gap-2">
            <input
              id="etc-default"
              type="checkbox"
              checked={form.is_default}
              onChange={(e) =>
                setForm({ ...form, is_default: e.target.checked })
              }
              className="h-4 w-4 rounded border-input"
            />
            <Label htmlFor="etc-default">Set as default</Label>
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

// ===========================================================================
// API Keys section
// ===========================================================================

function ApiKeysSection() {
  const { settings, isLoading, update } = useServerSettings();
  const [mapboxToken, setMapboxToken] = useState("");
  const [googleKey, setGoogleKey] = useState("");
  const [showMapbox, setShowMapbox] = useState(false);
  const [showGoogle, setShowGoogle] = useState(false);
  const [saving, setSaving] = useState(false);

  // Sync form state when settings load
  useEffect(() => {
    if (settings) {
      setMapboxToken(settings.mapbox_access_token || "");
      setGoogleKey(settings.google_maps_api_key || "");
    }
  }, [settings]);

  async function handleSave() {
    setSaving(true);
    try {
      await update({
        mapbox_access_token: mapboxToken,
        google_maps_api_key: googleKey,
      });
      toast.success("API keys saved");
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to save API keys"
      );
    } finally {
      setSaving(false);
    }
  }

  if (isLoading) {
    return (
      <section className="space-y-4">
        <h2 className="text-2xl font-semibold">API Keys</h2>
        <Skeleton className="h-24 w-full" />
      </section>
    );
  }

  return (
    <section className="space-y-4">
      <div>
        <h2 className="text-2xl font-semibold">API Keys</h2>
        <p className="text-sm text-muted-foreground mt-1">
          Enter provider API keys to use MapBox or Google Maps tile sources.
        </p>
      </div>

      <div className="rounded-md border p-4 space-y-4 max-w-xl">
        <div className="space-y-2">
          <Label htmlFor="ak-mapbox">MapBox Access Token</Label>
          <div className="relative">
            <Input
              id="ak-mapbox"
              type={showMapbox ? "text" : "password"}
              value={mapboxToken}
              onChange={(e) => setMapboxToken(e.target.value)}
              placeholder="pk.eyJ1Ijoi..."
              className="pr-10"
            />
            <button
              type="button"
              onClick={() => setShowMapbox((v) => !v)}
              className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              aria-label={showMapbox ? "Hide token" : "Show token"}
            >
              {showMapbox ? (
                <EyeOff className="size-4" />
              ) : (
                <Eye className="size-4" />
              )}
            </button>
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="ak-google">Google Maps API Key</Label>
          <div className="relative">
            <Input
              id="ak-google"
              type={showGoogle ? "text" : "password"}
              value={googleKey}
              onChange={(e) => setGoogleKey(e.target.value)}
              placeholder="AIzaSy..."
              className="pr-10"
            />
            <button
              type="button"
              onClick={() => setShowGoogle((v) => !v)}
              className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              aria-label={showGoogle ? "Hide key" : "Show key"}
            >
              {showGoogle ? (
                <EyeOff className="size-4" />
              ) : (
                <Eye className="size-4" />
              )}
            </button>
          </div>
        </div>

        <Button onClick={handleSave} disabled={saving}>
          {saving ? "Saving..." : "Save"}
        </Button>
      </div>
    </section>
  );
}
