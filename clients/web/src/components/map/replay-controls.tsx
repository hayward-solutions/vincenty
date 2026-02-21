"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";

interface ReplayControlsProps {
  from: Date;
  to: Date;
  /** Called when the playback cursor time changes */
  onTimeChange: (time: Date) => void;
  /** Called when replay is stopped / reset */
  onReset: () => void;
}

/**
 * ReplayControls provides a time slider and play/pause/speed controls
 * for replaying location history tracks.
 */
export function ReplayControls({
  from,
  to,
  onTimeChange,
  onReset,
}: ReplayControlsProps) {
  const [isPlaying, setIsPlaying] = useState(false);
  const [speed, setSpeed] = useState(1);
  const [progress, setProgress] = useState(100); // 0-100
  const animRef = useRef<number | null>(null);
  const lastTickRef = useRef<number>(0);
  const progressRef = useRef(100);

  const totalMs = to.getTime() - from.getTime();

  const updateTime = useCallback(
    (pct: number) => {
      const clamped = Math.min(100, Math.max(0, pct));
      setProgress(clamped);
      progressRef.current = clamped;
      const ms = (clamped / 100) * totalMs;
      onTimeChange(new Date(from.getTime() + ms));
    },
    [from, totalMs, onTimeChange]
  );

  // Animation loop
  useEffect(() => {
    if (!isPlaying) {
      if (animRef.current) {
        cancelAnimationFrame(animRef.current);
        animRef.current = null;
      }
      return;
    }

    lastTickRef.current = performance.now();

    const tick = (now: number) => {
      const delta = now - lastTickRef.current;
      lastTickRef.current = now;

      // speed multiplier: at 1x, 1 second of real time = 1 minute of replay
      const replayDelta = delta * speed * 60;
      const pctDelta = (replayDelta / totalMs) * 100;
      const newProgress = progressRef.current + pctDelta;

      if (newProgress >= 100) {
        updateTime(100);
        setIsPlaying(false);
        return;
      }

      updateTime(newProgress);
      animRef.current = requestAnimationFrame(tick);
    };

    animRef.current = requestAnimationFrame(tick);

    return () => {
      if (animRef.current) cancelAnimationFrame(animRef.current);
    };
  }, [isPlaying, speed, totalMs, updateTime]);

  const handleSliderChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = parseFloat(e.target.value);
    updateTime(val);
  };

  const handlePlayPause = () => {
    if (progress >= 100) {
      updateTime(0);
    }
    setIsPlaying(!isPlaying);
  };

  const cycleSpeed = () => {
    const speeds = [1, 2, 5, 10];
    const idx = speeds.indexOf(speed);
    setSpeed(speeds[(idx + 1) % speeds.length]);
  };

  const currentTime = new Date(from.getTime() + (progress / 100) * totalMs);

  return (
    <div className="absolute bottom-4 left-4 right-4 z-10 flex flex-wrap items-center gap-2 sm:gap-3 bg-card/90 backdrop-blur-sm border rounded-lg px-3 py-2 sm:px-4 sm:py-3 shadow-lg">
      <Button
        variant="ghost"
        size="sm"
        onClick={handlePlayPause}
        className="shrink-0 w-8 h-8 p-0"
      >
        {isPlaying ? "||" : progress >= 100 ? "R" : "\u25B6"}
      </Button>

      <Button
        variant="ghost"
        size="sm"
        onClick={cycleSpeed}
        className="shrink-0 text-xs w-10 h-8 p-0"
      >
        {speed}x
      </Button>

      <input
        type="range"
        min={0}
        max={100}
        step={0.1}
        value={progress}
        onChange={handleSliderChange}
        className="flex-1 h-1.5 accent-primary cursor-pointer"
      />

      <span className="text-xs text-muted-foreground shrink-0 w-20 sm:w-32 text-right font-mono">
        {formatTime(currentTime)}
      </span>

      <Button
        variant="ghost"
        size="sm"
        onClick={onReset}
        className="shrink-0 text-xs h-8"
      >
        Close
      </Button>
    </div>
  );
}

function formatTime(d: Date): string {
  return d.toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}
