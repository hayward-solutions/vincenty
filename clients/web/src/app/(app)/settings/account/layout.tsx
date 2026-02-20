"use client";

import { SettingsSidebar } from "@/components/settings/settings-sidebar";

const accountItems = [
  { href: "/settings/account/general", label: "General" },
  { href: "/settings/account/security", label: "Security" },
  { href: "/settings/account/devices", label: "Devices" },
  { href: "/settings/account/activity", label: "Activity" },
];

export default function AccountSettingsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex h-[calc(100vh-3.5rem)]">
      <SettingsSidebar title="Account Settings" items={accountItems} />
      <div className="flex-1 overflow-auto">{children}</div>
    </div>
  );
}
