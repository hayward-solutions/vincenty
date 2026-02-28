"use client";

import Link from "next/link";
import { Users, ArrowRight } from "lucide-react";
import { useMyGroups } from "@/lib/hooks/use-location-history";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";

export function GroupsList() {
  const { groups, isLoading } = useMyGroups();

  const LIMIT = 6;
  const shown = groups.slice(0, LIMIT);

  return (
    <Card className="flex flex-col">
      <CardHeader className="flex flex-row items-center justify-between pb-3">
        <CardTitle className="text-base font-semibold">My Groups</CardTitle>
        <Button asChild variant="ghost" size="sm" className="text-xs gap-1">
          <Link href="/messages">
            View all
            <ArrowRight className="h-3 w-3" />
          </Link>
        </Button>
      </CardHeader>
      <CardContent className="flex-1 p-0">
        {isLoading ? (
          <ul className="divide-y">
            {Array.from({ length: 4 }).map((_, i) => (
              <li key={i} className="flex items-center gap-3 px-6 py-3">
                <Skeleton className="h-7 w-7 rounded-full shrink-0" />
                <div className="flex-1 min-w-0 space-y-1">
                  <Skeleton className="h-3.5 w-28" />
                  <Skeleton className="h-3 w-20" />
                </div>
                <Skeleton className="h-3 w-12 shrink-0" />
              </li>
            ))}
          </ul>
        ) : groups.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-10 text-center px-6">
            <Users className="h-8 w-8 text-muted-foreground mb-2" />
            <p className="text-sm text-muted-foreground">
              You are not a member of any groups.
            </p>
          </div>
        ) : (
          <ul className="divide-y">
            {shown.map((group) => (
              <li key={group.id}>
                <Link
                  href={`/messages?group=${group.id}`}
                  className="flex items-center gap-3 px-6 py-3 hover:bg-muted/50 transition-colors"
                >
                  {/* Color swatch */}
                  <span
                    className="h-7 w-7 rounded-full shrink-0 border border-border"
                    style={{ background: group.marker_color || "#6b7280" }}
                    aria-hidden
                  />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">{group.name}</p>
                    {group.description ? (
                      <p className="text-xs text-muted-foreground truncate">
                        {group.description}
                      </p>
                    ) : null}
                  </div>
                  <span className="text-xs text-muted-foreground whitespace-nowrap">
                    {group.member_count}{" "}
                    {group.member_count === 1 ? "member" : "members"}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
