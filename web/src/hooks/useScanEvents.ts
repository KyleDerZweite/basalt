import { useCallback, useEffect, useRef, useState } from "react";
import type { ScanEvent } from "../types";
import { scanEventTypes, ACTIVE_STATUSES } from "../lib/constants";

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
  stableUpdate.current = onWorkspaceUpdate;

  // Initial fetch for history
  useEffect(() => {
    if (!scanId) return;
    const fetchHistory = async () => {
      try {
        const resp = await fetch(`/api/scans/${scanId}/events`);
        const data = await resp.json() as { events: ScanEvent[] };
        setEvents(data.events.reverse()); // backlog is usually oldest first, we want newest-first
      } catch (err) {
        console.error("Failed to fetch event history:", err);
      }
    };
    void fetchHistory();
  }, [scanId]);

  const isActive = (ACTIVE_STATUSES as readonly string[]).includes(scanStatus);

  // SSE event stream
  useEffect(() => {
    if (!scanId) return;

    const es = new EventSource(`/api/scans/${scanId}/events?stream=1`);
    esRef.current = es;

    es.onopen = () => setIsConnected(true);
    es.onerror = () => setIsConnected(false);

    const handler = (ev: MessageEvent) => {
      try {
        const event = JSON.parse(ev.data as string) as ScanEvent;
        setEvents((prev) => {
          // keep newest-first, max 200
          const next = [event, ...prev];
          return next.length > 200 ? next.slice(0, 200) : next;
        });
        // trigger workspace refresh on every event
        stableUpdate.current();
      } catch {
        // ignore malformed events
      }
    };

    for (const type of scanEventTypes) {
      es.addEventListener(type, handler);
    }

    return () => {
      es.close();
      esRef.current = null;
      setIsConnected(false);
    };
  }, [scanId]);

  // Close SSE once scan is no longer active
  useEffect(() => {
    if (!isActive && esRef.current) {
      esRef.current.close();
      esRef.current = null;
      setIsConnected(false);
    }
  }, [isActive]);

  // Poll workspace every 3s while active
  const poll = useCallback(() => {
    stableUpdate.current();
  }, []);

  useEffect(() => {
    if (!isActive) return;
    const timer = setInterval(poll, 3000);
    return () => clearInterval(timer);
  }, [isActive, poll]);

  return { events, isConnected };
}
