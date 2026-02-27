"use client";

import { useEffect, useState } from "react";
import { Copy, Check } from "lucide-react";
import { toast } from "sonner";
import {
  useApiTokens,
  useCreateApiToken,
  useDeleteApiToken,
} from "@/lib/hooks/use-api-tokens";
import { ApiError } from "@/lib/api";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import type { CreateApiTokenResponse } from "@/types/api";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function relativeTime(iso?: string): string {
  if (!iso) return "Never";
  const seconds = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (seconds < 60) return "Just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo ago`;
  return `${Math.floor(months / 12)}y ago`;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function ApiTokenList() {
  const { tokens, isLoading, error, fetch } = useApiTokens();
  const { createToken, isLoading: isCreating } = useCreateApiToken();
  const { deleteToken } = useDeleteApiToken();

  // Create dialog state
  const [createOpen, setCreateOpen] = useState(false);
  const [tokenName, setTokenName] = useState("");
  const [tokenExpiry, setTokenExpiry] = useState("");

  // Reveal dialog state (shown once after creation)
  const [revealData, setRevealData] = useState<CreateApiTokenResponse | null>(
    null
  );
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    fetch();
  }, [fetch]);

  // ------- Create flow -------

  function openCreateDialog() {
    setTokenName("");
    setTokenExpiry("");
    setCreateOpen(true);
  }

  async function handleCreate() {
    const name = tokenName.trim();
    if (!name) {
      toast.error("Token name is required");
      return;
    }

    try {
      const result = await createToken({
        name,
        expires_at: tokenExpiry || undefined,
      });
      setCreateOpen(false);
      setRevealData(result);
      setCopied(false);
      fetch();
      toast.success("Token created");
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to create token"
      );
    }
  }

  // ------- Copy -------

  async function handleCopy() {
    if (!revealData) return;
    await navigator.clipboard.writeText(revealData.token);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  // ------- Revoke -------

  async function handleRevoke(id: string, name: string) {
    if (!confirm(`Revoke token "${name}"? This cannot be undone.`)) return;
    try {
      await deleteToken(id);
      fetch();
      toast.success("Token revoked");
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to revoke token"
      );
    }
  }

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>API Tokens</CardTitle>
          <CardDescription>
            Create tokens for CLI or API access. Tokens are shown once at
            creation time.
          </CardDescription>
          <CardAction>
            <Button size="sm" onClick={openCreateDialog}>
              Create Token
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent>
          {error && (
            <p className="text-sm text-destructive mb-4">{error}</p>
          )}

          {isLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 3 }).map((_, i) => {
                const key = `skeleton-${i}`;
                return <Skeleton key={key} className="h-12 w-full" />;
              })}
            </div>
          ) : (
            <div className="rounded-md border overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Last Used</TableHead>
                    <TableHead>Expires</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead className="w-[1%]" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {tokens.map((token) => (
                    <TableRow key={token.id}>
                      <TableCell className="font-medium">
                        {token.name}
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-sm text-muted-foreground">
                        {relativeTime(token.last_used_at)}
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-sm text-muted-foreground">
                        {token.expires_at
                          ? formatDate(token.expires_at)
                          : "Never"}
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-sm text-muted-foreground">
                        {formatDate(token.created_at)}
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive"
                          onClick={() => handleRevoke(token.id, token.name)}
                        >
                          Revoke
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                  {tokens.length === 0 && (
                    <TableRow>
                      <TableCell
                        colSpan={5}
                        className="text-center text-muted-foreground py-8"
                      >
                        No API tokens
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create token dialog */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create API Token</DialogTitle>
            <DialogDescription>
              Give this token a name to identify its purpose.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="token-name">Name</Label>
              <Input
                id="token-name"
                value={tokenName}
                onChange={(e) => setTokenName(e.target.value)}
                maxLength={100}
                placeholder="e.g. CI Pipeline, Field Device"
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleCreate();
                }}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="token-expiry">
                Expiry Date{" "}
                <span className="text-muted-foreground font-normal">
                  (optional)
                </span>
              </Label>
              <Input
                id="token-expiry"
                type="date"
                value={tokenExpiry}
                onChange={(e) => setTokenExpiry(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleCreate} disabled={isCreating}>
              {isCreating ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Token reveal dialog — shown once after creation */}
      <Dialog
        open={revealData !== null}
        onOpenChange={(open) => {
          if (!open) setRevealData(null);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Token Created</DialogTitle>
            <DialogDescription>
              Copy this token now. It won&apos;t be shown again.
            </DialogDescription>
          </DialogHeader>
          <div className="py-2">
            <div className="flex items-center gap-2">
              <code className="flex-1 rounded-md bg-muted px-3 py-2 text-sm font-mono break-all">
                {revealData?.token}
              </code>
              <Button
                variant="outline"
                size="icon"
                onClick={handleCopy}
                title="Copy to clipboard"
              >
                {copied ? (
                  <Check className="size-4" />
                ) : (
                  <Copy className="size-4" />
                )}
              </Button>
            </div>
          </div>
          <DialogFooter>
            <Button onClick={() => setRevealData(null)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
