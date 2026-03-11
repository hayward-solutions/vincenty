"use client";

import { useRef, useState } from "react";
import { useAuth } from "@/lib/auth-context";
import { useUpdateMe, useUploadAvatar, useDeleteAvatar } from "@/lib/hooks/use-profile";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";
import {
  AVAILABLE_SHAPES,
  MARKER_SHAPES,
  PRESET_COLORS,
  markerSVGString,
} from "@/components/map/marker-shapes";

export default function GeneralSettingsPage() {
  const { user, refreshUser } = useAuth();
  const { updateMe, isLoading: isUpdating } = useUpdateMe();
  const { uploadAvatar, isLoading: isUploading } = useUploadAvatar();
  const { deleteAvatar, isLoading: isDeleting } = useDeleteAvatar();

  const [displayName, setDisplayName] = useState(user?.display_name ?? "");
  const [email, setEmail] = useState(user?.email ?? "");
  const [markerIcon, setMarkerIcon] = useState(user?.marker_icon || "circle");
  const [markerColor, setMarkerColor] = useState(user?.marker_color || "#3b82f6");
  const [markerCustomColor, setMarkerCustomColor] = useState(
    PRESET_COLORS.includes(user?.marker_color || "#3b82f6")
      ? ""
      : user?.marker_color || ""
  );
  const [isSavingMarker, setIsSavingMarker] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const initials = user?.display_name
    ? user.display_name
        .split(" ")
        .map((n) => n[0])
        .join("")
        .toUpperCase()
        .slice(0, 2)
    : user?.username?.slice(0, 2).toUpperCase() ?? "??";

  const avatarSrc = user?.avatar_url
    ? `/api/v1/users/${user.id}/avatar?token=${typeof window !== "undefined" ? localStorage.getItem("access_token") ?? "" : ""}`
    : undefined;

  async function handleProfileSave(e: React.FormEvent) {
    e.preventDefault();
    try {
      await updateMe({
        display_name: displayName || undefined,
        email: email || undefined,
      });
      await refreshUser();
      toast.success("Profile updated");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to update profile");
    }
  }

  async function handleMarkerSave() {
    setIsSavingMarker(true);
    try {
      await updateMe({
        marker_icon: markerIcon,
        marker_color: markerColor,
      });
      await refreshUser();
      toast.success("Map marker updated");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to update marker");
    } finally {
      setIsSavingMarker(false);
    }
  }

  function handleMarkerCustomColorChange(value: string) {
    setMarkerCustomColor(value);
    if (/^#[0-9a-fA-F]{6}$/.test(value)) {
      setMarkerColor(value);
    }
  }

  async function handleAvatarUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;

    // Client-side validation
    const allowedTypes = ["image/jpeg", "image/png", "image/webp"];
    if (!allowedTypes.includes(file.type)) {
      toast.error("Please select a JPEG, PNG, or WebP image");
      return;
    }
    if (file.size > 5 * 1024 * 1024) {
      toast.error("Image must be smaller than 5 MB");
      return;
    }

    try {
      await uploadAvatar(file);
      await refreshUser();
      toast.success("Avatar updated");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to upload avatar");
    } finally {
      // Reset file input so the same file can be re-selected
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  }

  async function handleAvatarRemove() {
    try {
      await deleteAvatar();
      await refreshUser();
      toast.success("Avatar removed");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to remove avatar");
    }
  }

  return (
    <div className="p-4 md:p-6 space-y-6 max-w-2xl">
      <h1 className="text-2xl font-semibold">General</h1>

      {/* Avatar Card */}
      <Card>
        <CardHeader>
          <CardTitle>Avatar</CardTitle>
          <CardDescription>
            Upload a profile picture. JPEG, PNG, or WebP up to 5 MB.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4">
            <Avatar className="h-20 w-20">
              {avatarSrc && <AvatarImage src={avatarSrc} alt="Avatar" />}
              <AvatarFallback className="text-xl">{initials}</AvatarFallback>
            </Avatar>
            <div className="flex flex-col gap-2">
              <input
                ref={fileInputRef}
                type="file"
                accept="image/jpeg,image/png,image/webp"
                className="hidden"
                onChange={handleAvatarUpload}
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={isUploading}
                onClick={() => fileInputRef.current?.click()}
              >
                {isUploading ? "Uploading..." : "Upload"}
              </Button>
              {user?.avatar_url && (
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  disabled={isDeleting}
                  onClick={handleAvatarRemove}
                >
                  {isDeleting ? "Removing..." : "Remove"}
                </Button>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Map Marker Card */}
      <Card>
        <CardHeader>
          <CardTitle>Map Marker</CardTitle>
          <CardDescription>
            Customize how your position appears on the map. Choose a shape and
            color for your &ldquo;You&rdquo; marker.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          {/* Live preview */}
          <div className="flex items-center justify-center p-4 bg-muted/50 rounded-lg">
            <div className="flex flex-col items-center gap-1">
              <span
                dangerouslySetInnerHTML={{
                  __html: markerSVGString(markerIcon, markerColor, 36),
                }}
              />
              <span className="text-xs text-muted-foreground mt-1">
                Preview
              </span>
            </div>
          </div>

          {/* Shape picker */}
          <div className="space-y-2">
            <Label>Shape</Label>
            <div className="grid grid-cols-5 gap-2">
              {AVAILABLE_SHAPES.map((shape) => (
                <button
                  key={shape}
                  type="button"
                  onClick={() => setMarkerIcon(shape)}
                  className={`flex flex-col items-center gap-1 p-2 rounded-md border-2 transition-colors ${
                    markerIcon === shape
                      ? "border-primary bg-primary/10"
                      : "border-transparent hover:bg-muted"
                  }`}
                >
                  <span
                    dangerouslySetInnerHTML={{
                      __html: markerSVGString(shape, markerColor, 20),
                    }}
                  />
                  <span className="text-[10px] text-muted-foreground">
                    {MARKER_SHAPES[shape].label}
                  </span>
                </button>
              ))}
            </div>
          </div>

          {/* Color picker */}
          <div className="space-y-2">
            <Label>Color</Label>
            <div className="flex flex-wrap gap-2">
              {PRESET_COLORS.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => {
                    setMarkerColor(c);
                    setMarkerCustomColor("");
                  }}
                  className={`w-7 h-7 rounded-full border-2 transition-all ${
                    markerColor === c && !markerCustomColor
                      ? "border-foreground scale-110"
                      : "border-transparent hover:scale-105"
                  }`}
                  style={{ backgroundColor: c }}
                  title={c}
                />
              ))}
            </div>
            <div className="flex items-center gap-2 mt-2">
              <Label htmlFor="marker-custom-color" className="text-xs whitespace-nowrap">
                Custom hex:
              </Label>
              <Input
                id="marker-custom-color"
                value={markerCustomColor}
                onChange={(e) => handleMarkerCustomColorChange(e.target.value)}
                placeholder="#ff0000"
                className="h-8 text-sm font-mono w-28"
                maxLength={7}
              />
              {markerCustomColor && /^#[0-9a-fA-F]{6}$/.test(markerCustomColor) && (
                <div
                  className="w-6 h-6 rounded-full border"
                  style={{ backgroundColor: markerCustomColor }}
                />
              )}
            </div>
          </div>

          <Button
            type="button"
            onClick={handleMarkerSave}
            disabled={isSavingMarker}
          >
            {isSavingMarker ? "Saving..." : "Save Marker"}
          </Button>
        </CardContent>
      </Card>

      {/* Profile Card */}
      <Card>
        <CardHeader>
          <CardTitle>Profile</CardTitle>
          <CardDescription>
            Update your display name and email address.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleProfileSave} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">Username</Label>
              <Input id="username" value={user?.username ?? ""} disabled />
              <p className="text-xs text-muted-foreground">
                Usernames cannot be changed.
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="displayName">Display Name</Label>
              <Input
                id="displayName"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder="Enter your display name"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="Enter your email"
              />
            </div>
            <Button type="submit" disabled={isUpdating}>
              {isUpdating ? "Saving..." : "Save Changes"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
