import { useCallback, useEffect, useRef, useState } from "react";
import type { ScanEvent } from "../types";
import { scanEventTypes, ACTIVE_STATUSES } from "../lib/constants";
import { api } from "../lib/api";

interface UseScanEventsOptions {
  scanId: string;
  scanStatus: string;
  onWorkspaceUpdate: () => void;
}

export function useScanEvents({ scanId, scanStatus, onWorkspaceUpdate }: UseScanEventsOptions) {
  const [events, setEvents] = useState<ScanEvent[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const esRef = useRef<EventSource | null>(null);
  const stableUpdate = useRef(onWorkspaceUpdate);
  const lastSequence = useRef(0);
  const refreshTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  stableUpdate.current = onWorkspaceUpdate;

  const isActive = (ACTIVE_STATUSES as readonly string[]).includes(scanStatus);

  useEffect(() => {
    if (!scanId) return;

    let disposed = false;
    const abortController = new AbortController();

    const scheduleWorkspaceUpdate = () => {
      if (refreshTimer.current !== null) return;
      refreshTimer.current = setTimeout(() => {
        refreshTimer.current = null;
        stableUpdate.current();
      }, 250);
    };

    const connect = async () => {
      lastSequence.current = 0;
      setEvents([]);
      setIsConnected(false);

      try {
        const data = await api<{ events: ScanEvent[] }>(`/api/scans/${scanId}/events`, {
          signal: abortController.signal,
        });
        if (disposed) return;

        const history = data.events.sort((a, b) => b.sequence - a.sequence).slice(0, 200);
        lastSequence.current = history[0]?.sequence ?? 0;
        setEvents(history);
      } catch (err) {
        if (!abortController.signal.aborted) {
          console.error("Failed to fetch event history:", err);
        }
        return;
      }

      if (!isActive || disposed) return;

      const es = new EventSource(`/api/scans/${scanId}/events?stream=1&after=${lastSequence.current}`);
      esRef.current = es;
      es.onopen = () => setIsConnected(true);
      es.onerror = () => setIsConnected(false);

      const handler = (ev: MessageEvent) => {
        try {
          const event = JSON.parse(ev.data as string) as ScanEvent;
          if (event.sequence <= lastSequence.current) return;
          lastSequence.current = event.sequence;
          setEvents((previous) => [
            event,
            ...previous.filter((item) => item.sequence !== event.sequence),
          ].slice(0, 200));
          scheduleWorkspaceUpdate();
        } catch {
          // Ignore malformed events and keep the stream alive.
        }
      };

      for (const type of scanEventTypes) {
        es.addEventListener(type, handler);
      }
    };

    void connect();

    return () => {
      disposed = true;
      abortController.abort();
      esRef.current?.close();
      esRef.current = null;
      if (refreshTimer.current !== null) {
        clearTimeout(refreshTimer.current);
        refreshTimer.current = null;
      }
      setIsConnected(false);
    };
  }, [scanId, isActive]);

  const poll = useCallback(() => {
    stableUpdate.current();
  }, []);

  useEffect(() => {
    if (!isActive) return;
    const timer = setInterval(poll, 15000);
    return () => clearInterval(timer);
  }, [isActive, poll]);

  return { events, isConnected };
}
