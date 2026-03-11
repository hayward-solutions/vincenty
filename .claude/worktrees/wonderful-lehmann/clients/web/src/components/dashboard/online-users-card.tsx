"use client";

import { Radio } from "lucide-react";
import { useLocations } from "@/lib/hooks/use-locations";
import { StatCard } from "@/components/dashboard/stat-card";

/**
 * Displays a live count of unique users currently broadcasting their location
 * over the WebSocket. Updates in real-time as location_broadcast and
 * location_snapshot events arrive.
 */
export function OnlineUsersCard() {
  const { locations } = useLocations();

  // Count unique user IDs (a user may have multiple devices)
  const uniqueUsers = new Set(
    Array.from(locations.values()).map((loc) => loc.user_id)
  ).size;

  return (
    <StatCard
      title="Online Now"
      value={uniqueUsers}
      icon={Radio}
      description={uniqueUsers === 1 ? "user reporting location" : "users reporting location"}
      iconClassName="text-green-500"
    />
  );
}
