"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { SettingsSidebar } from "@/components/settings/settings-sidebar";

const serverItems = [
  { href: "/settings/server/map", label: "Map" },
  { href: "/settings/server/users", label: "Users" },
  { href: "/settings/server/groups", label: "Groups" },
  { href: "/settings/server/security", label: "Security" },
  { href: "/settings/server/audit-logs", label: "Audit Logs" },
];

export default function ServerSettingsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();
  const { isAdmin, isLoading } = useAuth();

  useEffect(() => {
    if (!isLoading && !isAdmin) {
      router.push("/dashboard");
    }
  }, [isLoading, isAdmin, router]);

  if (isLoading || !isAdmin) {
    return null;
  }

  return (
    <div className="flex flex-col md:flex-row h-[calc(100vh-3.5rem)]">
      <SettingsSidebar title="Server Settings" items={serverItems} />
      <div className="flex-1 overflow-auto">{children}</div>
    </div>
  );
}
