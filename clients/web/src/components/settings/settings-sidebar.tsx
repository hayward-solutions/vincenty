"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Button } from "@/components/ui/button";

interface SettingsSidebarItem {
  href: string;
  label: string;
}

interface SettingsSidebarProps {
  title: string;
  items: SettingsSidebarItem[];
}

export function SettingsSidebar({ title, items }: SettingsSidebarProps) {
  const pathname = usePathname();

  return (
    <nav className="w-56 shrink-0 border-r p-4 space-y-1">
      <h2 className="px-3 mb-2 text-lg font-semibold tracking-tight">
        {title}
      </h2>
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
    </nav>
  );
}
