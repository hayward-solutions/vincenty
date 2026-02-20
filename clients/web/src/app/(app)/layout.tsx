"use client";

import { useEffect } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { useAuth } from "@/lib/auth-context";
import { useWebSocket } from "@/lib/websocket-context";
import { useLocationSharing } from "@/lib/hooks/use-location-sharing";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const { user, isLoading, isAuthenticated, isAdmin, logout } = useAuth();
  const { connectionState } = useWebSocket();
  const { error: locationError } = useLocationSharing();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.push("/login");
    }
  }, [isLoading, isAuthenticated, router]);

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Skeleton className="h-8 w-48" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return null;
  }

  const initials = user?.display_name
    ? user.display_name
        .split(" ")
        .map((n) => n[0])
        .join("")
        .toUpperCase()
        .slice(0, 2)
    : user?.username?.slice(0, 2).toUpperCase() ?? "??";

  const navItems = [
    { href: "/dashboard", label: "Dashboard" },
    { href: "/map", label: "Map" },
    { href: "/messages", label: "Messages" },
  ];

  async function handleLogout() {
    await logout();
    router.push("/login");
  }

  return (
    <div className="flex min-h-screen flex-col">
      <header className="border-b">
        <div className="flex h-14 items-center px-4 gap-4">
          <Link href="/dashboard" className="font-semibold text-lg">
            SitAware
          </Link>
          <Separator orientation="vertical" className="h-6" />
          <nav className="flex items-center gap-1">
            {navItems.map((item) => (
              <Button
                key={item.href}
                variant={pathname === item.href ? "secondary" : "ghost"}
                size="sm"
                asChild
              >
                <Link href={item.href}>{item.label}</Link>
              </Button>
            ))}
          </nav>
          <div className="ml-auto flex items-center gap-3">
            <div
              className="flex items-center gap-1.5 text-xs px-2 py-1"
              title={
                connectionState !== "connected"
                  ? "Cannot connect to server"
                  : locationError
                    ? locationError
                    : "All systems operational"
              }
            >
              <span
                className={`inline-block h-2 w-2 rounded-full ${
                  connectionState !== "connected"
                    ? "bg-red-500"
                    : locationError
                      ? "bg-yellow-500"
                      : "bg-green-500"
                }`}
              />
              <span className="text-muted-foreground hidden sm:inline">
                {connectionState !== "connected"
                  ? "Error"
                  : locationError
                    ? "Degraded"
                    : "Connected"}
              </span>
            </div>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  className="relative h-8 w-8 rounded-full"
                >
                  <Avatar className="h-8 w-8">
                    <AvatarFallback className="text-xs">
                      {initials}
                    </AvatarFallback>
                  </Avatar>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <div className="flex items-center gap-2 p-2">
                  <div className="flex flex-col space-y-0.5">
                    <p className="text-sm font-medium">{user?.display_name || user?.username}</p>
                    <p className="text-xs text-muted-foreground">
                      {user?.email}
                    </p>
                  </div>
                </div>
                <DropdownMenuSeparator />
                <DropdownMenuItem asChild>
                  <Link href="/settings/account">Account Settings</Link>
                </DropdownMenuItem>
                {isAdmin && (
                  <DropdownMenuItem asChild>
                    <Link href="/settings/server">Server Settings</Link>
                  </DropdownMenuItem>
                )}
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={handleLogout}>
                  Sign out
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </header>
      <main className="flex-1">{children}</main>
    </div>
  );
}
