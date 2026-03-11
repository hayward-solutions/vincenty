"use client";

import { useEffect } from "react";
import { Map, MessageSquare, Users, Smartphone, Shield, LayoutGrid } from "lucide-react";
import { useAuth } from "@/lib/auth-context";
import { useConversations } from "@/lib/hooks/use-conversations";
import { useMyGroups } from "@/lib/hooks/use-location-history";
import { useMyDevices } from "@/lib/hooks/use-devices";
import { useUsers } from "@/lib/hooks/use-users";
import { useGroups } from "@/lib/hooks/use-groups";
import { useAllLocations } from "@/lib/hooks/use-location-history";
import { StatCard } from "@/components/dashboard/stat-card";
import { OnlineUsersCard } from "@/components/dashboard/online-users-card";
import { RecentMessages } from "@/components/dashboard/recent-messages";
import { GroupsList } from "@/components/dashboard/groups-list";
import { RecentActivity } from "@/components/dashboard/recent-activity";
import { DevicesPanel } from "@/components/dashboard/devices-panel";
import { Badge } from "@/components/ui/badge";

// ---------------------------------------------------------------------------
// Admin-only stat cards — rendered conditionally
// ---------------------------------------------------------------------------

function AdminStats() {
  const { data: usersData, isLoading: usersLoading } = useUsers(1, 1);
  const { data: groupsData, isLoading: groupsLoading } = useGroups(1, 1);
  const { data: allLocations, isLoading: locLoading, fetchAll } = useAllLocations();

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  return (
    <>
      <StatCard
        title="Total Users"
        value={usersData?.total}
        icon={Shield}
        description="registered accounts"
        isLoading={usersLoading}
      />
      <StatCard
        title="Total Groups"
        value={groupsData?.total}
        icon={LayoutGrid}
        description="configured groups"
        isLoading={groupsLoading}
      />
      <StatCard
        title="Active Trackers"
        value={allLocations.length}
        icon={Map}
        description="devices with a known location"
        isLoading={locLoading}
      />
    </>
  );
}

// ---------------------------------------------------------------------------
// Dashboard page
// ---------------------------------------------------------------------------

export default function DashboardPage() {
  const { user, isAdmin } = useAuth();
  const { conversations, isLoading: convsLoading } = useConversations();
  const { groups, isLoading: groupsLoading } = useMyGroups();
  const { devices, isLoading: devicesLoading, fetch: fetchDevices } = useMyDevices();

  useEffect(() => {
    fetchDevices();
  }, [fetchDevices]);

  return (
    <div className="p-6 space-y-6">
      {/* Page heading */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Dashboard</h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            Welcome back,{" "}
            <span className="font-medium text-foreground">
              {user?.display_name || user?.username}
            </span>
            {isAdmin && (
              <Badge variant="secondary" className="ml-2 text-xs">
                Admin
              </Badge>
            )}
          </p>
        </div>
      </div>

      {/* ── Stat cards ─────────────────────────────────────────────────── */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {/* Live online count — always visible */}
        <OnlineUsersCard />

        <StatCard
          title="My Groups"
          value={groupsLoading ? undefined : groups.length}
          icon={Users}
          description={groups.length === 1 ? "group membership" : "group memberships"}
          isLoading={groupsLoading}
        />

        <StatCard
          title="Conversations"
          value={convsLoading ? undefined : conversations.length}
          icon={MessageSquare}
          description={
            conversations.length === 1 ? "active conversation" : "active conversations"
          }
          isLoading={convsLoading}
        />

        <StatCard
          title="Devices"
          value={devicesLoading ? undefined : devices.length}
          icon={Smartphone}
          description={
            devices.length === 1 ? "registered device" : "registered devices"
          }
          isLoading={devicesLoading}
        />
      </div>

      {/* ── Admin stat cards ────────────────────────────────────────────── */}
      {isAdmin && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <AdminStats />
        </div>
      )}

      {/* ── Middle row: messages + groups ──────────────────────────────── */}
      <div className="grid gap-4 lg:grid-cols-2">
        <RecentMessages />
        <GroupsList />
      </div>

      {/* ── Bottom row: activity + devices ─────────────────────────────── */}
      <div className="grid gap-4 lg:grid-cols-2">
        <RecentActivity />
        <DevicesPanel />
      </div>
    </div>
  );
}
