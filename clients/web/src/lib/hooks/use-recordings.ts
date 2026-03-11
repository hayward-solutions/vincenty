"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { Recording } from "@/types/api";

export function useStreamRecordings(streamId: string) {
  const [recordings, setRecordings] = useState<Recording[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchRecordings = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api.get<Recording[]>(
        `/api/v1/streams/${streamId}/recordings`
      );
      setRecordings(result);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch recordings"
      );
    } finally {
      setIsLoading(false);
    }
  }, [streamId]);

  useEffect(() => {
    fetchRecordings();
  }, [fetchRecordings]);

  return { recordings, isLoading, error, refetch: fetchRecordings };
}

export function useStartRecording() {
  const [isLoading, setIsLoading] = useState(false);

  const startRecording = async (streamId: string): Promise<Recording> => {
    setIsLoading(true);
    try {
      return await api.post<Recording>(
        `/api/v1/streams/${streamId}/recordings/start`
      );
    } finally {
      setIsLoading(false);
    }
  };

  return { startRecording, isLoading };
}

export function useStopRecording() {
  const [isLoading, setIsLoading] = useState(false);

  const stopRecording = async (recordingId: string): Promise<Recording> => {
    setIsLoading(true);
    try {
      return await api.post<Recording>(
        `/api/v1/recordings/${recordingId}/stop`
      );
    } finally {
      setIsLoading(false);
    }
  };

  return { stopRecording, isLoading };
}
