"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { AuditFilters } from "@/types/api";

const ACTIONS = [
  { value: "", label: "All Actions" },
  { value: "auth.login", label: "Login" },
  { value: "auth.logout", label: "Logout" },
  { value: "user.create", label: "Create User" },
  { value: "user.update", label: "Update User" },
  { value: "user.delete", label: "Delete User" },
  { value: "group.create", label: "Create Group" },
  { value: "group.update", label: "Update Group" },
  { value: "group.delete", label: "Delete Group" },
  { value: "group.member_add", label: "Add Member" },
  { value: "group.member_remove", label: "Remove Member" },
  { value: "message.send", label: "Send Message" },
  { value: "message.delete", label: "Delete Message" },
  { value: "map_config.create", label: "Create Map Config" },
  { value: "map_config.update", label: "Update Map Config" },
  { value: "map_config.delete", label: "Delete Map Config" },
];

const RESOURCE_TYPES = [
  { value: "", label: "All Types" },
  { value: "session", label: "Session" },
  { value: "user", label: "User" },
  { value: "device", label: "Device" },
  { value: "group", label: "Group" },
  { value: "message", label: "Message" },
  { value: "map_config", label: "Map Config" },
];

interface AuditFilterBarProps {
  onApply: (filters: AuditFilters) => void;
}

export function AuditFilterBar({ onApply }: AuditFilterBarProps) {
  const [action, setAction] = useState("");
  const [resourceType, setResourceType] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");

  function handleApply() {
    const filters: AuditFilters = {};
    if (action) filters.action = action;
    if (resourceType) filters.resource_type = resourceType;
    if (from) filters.from = new Date(from).toISOString();
    if (to) filters.to = new Date(to).toISOString();
    onApply(filters);
  }

  function handleReset() {
    setAction("");
    setResourceType("");
    setFrom("");
    setTo("");
    onApply({});
  }

  return (
    <div className="flex flex-wrap items-end gap-3">
      <div className="space-y-1">
        <Label className="text-xs">Action</Label>
        <select
          value={action}
          onChange={(e) => setAction(e.target.value)}
          className="h-9 rounded-md border border-input bg-background px-3 text-sm"
        >
          {ACTIONS.map((a) => (
            <option key={a.value} value={a.value}>
              {a.label}
            </option>
          ))}
        </select>
      </div>
      <div className="space-y-1">
        <Label className="text-xs">Resource Type</Label>
        <select
          value={resourceType}
          onChange={(e) => setResourceType(e.target.value)}
          className="h-9 rounded-md border border-input bg-background px-3 text-sm"
        >
          {RESOURCE_TYPES.map((t) => (
            <option key={t.value} value={t.value}>
              {t.label}
            </option>
          ))}
        </select>
      </div>
      <div className="space-y-1">
        <Label className="text-xs">From</Label>
        <Input
          type="datetime-local"
          value={from}
          onChange={(e) => setFrom(e.target.value)}
          className="h-9 w-48"
        />
      </div>
      <div className="space-y-1">
        <Label className="text-xs">To</Label>
        <Input
          type="datetime-local"
          value={to}
          onChange={(e) => setTo(e.target.value)}
          className="h-9 w-48"
        />
      </div>
      <Button size="sm" onClick={handleApply}>
        Filter
      </Button>
      <Button size="sm" variant="outline" onClick={handleReset}>
        Reset
      </Button>
    </div>
  );
}
