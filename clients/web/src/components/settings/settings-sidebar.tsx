"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Menu } from "lucide-react";

interface SettingsSidebarItem {
  href: string;
  label: string;
}

interface SettingsSidebarProps {
  title: string;
  items: SettingsSidebarItem[];
}

function SidebarNav({ items, pathname }: { items: SettingsSidebarItem[]; pathname: string }) {
  return (
    <>
      {items.map((item) => {
        const isActive =
          pathname === item.href || pathname.startsWith(item.href + "/");
        return (
          <Button
            key={item.href}
            variant={isActive ? "secondary" : "ghost"}
            className="w-full justify-start"
            size="sm"
            asChild
          >
            <Link href={item.href}>{item.label}</Link>
          </Button>
        );
      })}
    </>
  );
}

export function SettingsSidebar({ title, items }: SettingsSidebarProps) {
  const pathname = usePathname();
  const [open, setOpen] = useState(false);

  // Close sheet on navigation
  const currentPath = pathname;

  return (
    <>
      {/* Mobile: menu button rendered in flow above the content */}
      <div className="md:hidden flex items-center gap-2 border-b px-4 py-2">
        <Button
          variant="ghost"
          size="sm"
          className="h-8 w-8 p-0"
          onClick={() => setOpen(true)}
        >
          <Menu className="h-4 w-4" />
          <span className="sr-only">Open settings menu</span>
        </Button>
        <span className="text-sm font-semibold">{title}</span>
      </div>

      {/* Mobile: sidebar as sheet */}
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent side="left" className="w-64 p-0">
          <SheetHeader className="px-4 py-3 border-b">
            <SheetTitle>{title}</SheetTitle>
          </SheetHeader>
          <nav className="p-3 space-y-1" onClick={() => setOpen(false)}>
            <SidebarNav items={items} pathname={currentPath} />
          </nav>
        </SheetContent>
      </Sheet>

      {/* Desktop: static sidebar */}
      <nav className="hidden md:block w-56 shrink-0 border-r p-4 space-y-1">
        <h2 className="px-3 mb-2 text-lg font-semibold tracking-tight">
          {title}
        </h2>
        <SidebarNav items={items} pathname={currentPath} />
      </nav>
    </>
  );
}
