"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { useAuth } from "@/lib/auth-context";
import { useWebSocket } from "@/lib/websocket-context";
import { useLocationSharing } from "@/lib/hooks/use-location-sharing";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Menu } from "lucide-react";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const { user, isLoading, isAuthenticated, isAdmin, logout } = useAuth();
  const { connectionState } = useWebSocket();
  const { error: locationError } = useLocationSharing();
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.push("/login");
    }
  }, [isLoading, isAuthenticated, router]);

  // Close mobile nav on route change
  useEffect(() => {
    setMobileNavOpen(false);
  }, [pathname]);

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

  const avatarSrc = user?.avatar_url
    ? `/api/v1/users/${user.id}/avatar?token=${typeof window !== "undefined" ? localStorage.getItem("access_token") ?? "" : ""}`
    : undefined;

  const navItems = [
    { href: "/dashboard", label: "Dashboard" },
    { href: "/map", label: "Map" },
    { href: "/messages", label: "Messages" },
    { href: "/media", label: "Media" },
  ];

  async function handleLogout() {
    await logout();
    router.push("/login");
  }

  const statusTitle =
    connectionState !== "connected"
      ? "Cannot connect to server"
      : locationError
        ? locationError
        : "All systems operational";

  const statusDotColor =
    connectionState !== "connected"
      ? "bg-red-500"
      : locationError
        ? "bg-yellow-500"
        : "bg-green-500";

  const statusLabel =
    connectionState !== "connected"
      ? "Error"
      : locationError
        ? "Degraded"
        : "Connected";

  return (
    <div className="flex min-h-screen flex-col">
      <header className="border-b">
        <div className="flex h-14 items-center px-4 gap-4">
          {/* Mobile hamburger */}
          <Button
            variant="ghost"
            size="sm"
            className="md:hidden -ml-2 h-8 w-8 p-0"
            onClick={() => setMobileNavOpen(true)}
          >
            <Menu className="h-5 w-5" />
            <span className="sr-only">Open navigation</span>
          </Button>

          <Link href="/dashboard" className="font-semibold text-lg">
            SitAware
          </Link>
          <Separator orientation="vertical" className="h-6 hidden md:block" />

          {/* Desktop nav */}
          <nav className="hidden md:flex items-center gap-1">
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
              title={statusTitle}
            >
              <span
                className={`inline-block h-2 w-2 rounded-full ${statusDotColor}`}
              />
              <span className="text-muted-foreground hidden sm:inline">
                {statusLabel}
              </span>
            </div>

            {/* Desktop user menu */}
            <div className="hidden md:block">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    className="relative h-8 w-8 rounded-full"
                  >
                    <Avatar className="h-8 w-8">
                      {avatarSrc && <AvatarImage src={avatarSrc} alt="Avatar" />}
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
        </div>
      </header>

      {/* Mobile nav sheet */}
      <Sheet open={mobileNavOpen} onOpenChange={setMobileNavOpen}>
        <SheetContent side="left" className="w-64 p-0">
          <SheetHeader className="border-b px-4 py-3">
            <SheetTitle>SitAware</SheetTitle>
          </SheetHeader>
          <nav className="flex flex-col gap-1 p-3">
            {navItems.map((item) => (
              <Button
                key={item.href}
                variant={pathname === item.href ? "secondary" : "ghost"}
                className="w-full justify-start"
                asChild
              >
                <Link href={item.href}>{item.label}</Link>
              </Button>
            ))}
          </nav>
          <Separator />
          <div className="flex flex-col gap-1 p-3">
            <Button variant="ghost" className="w-full justify-start" asChild>
              <Link href="/settings/account">Account Settings</Link>
            </Button>
            {isAdmin && (
              <Button variant="ghost" className="w-full justify-start" asChild>
                <Link href="/settings/server">Server Settings</Link>
              </Button>
            )}
          </div>
          <div className="mt-auto border-t p-3">
            <div className="flex items-center gap-3 mb-3 px-2">
              <Avatar className="h-8 w-8">
                {avatarSrc && <AvatarImage src={avatarSrc} alt="Avatar" />}
                <AvatarFallback className="text-xs">{initials}</AvatarFallback>
              </Avatar>
              <div className="flex flex-col min-w-0">
                <p className="text-sm font-medium truncate">
                  {user?.display_name || user?.username}
                </p>
                <p className="text-xs text-muted-foreground truncate">
                  {user?.email}
                </p>
              </div>
            </div>
            <Button
              variant="outline"
              size="sm"
              className="w-full"
              onClick={handleLogout}
            >
              Sign out
            </Button>
          </div>
        </SheetContent>
      </Sheet>

      <main className="flex-1">{children}</main>
    </div>
  );
}
