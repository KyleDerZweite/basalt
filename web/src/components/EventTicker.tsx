import { RefreshCw } from "lucide-react";
import type { ScanEvent } from "../types";
import { PretextBlock } from "./PretextBlock";
import { lineHeights, pretextFonts } from "../lib/typography";

interface EventTickerProps {
  events: ScanEvent[];
  isConnected?: boolean;
}

function chipClass(type: string): string {
  if (type === "node_discovered" || type === "edge_discovered") return "node_discovered";
  if (type === "module_error" || type === "scan_failed") return "module_error";
  if (type === "scan_finished") return "scan_finished";
  if (type === "scan_started" || type === "module_started" || type === "verify_complete") return "scan_started";
  return "default";
}

function formatTime(iso: string): string {
  try {
    return new Date(iso).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit" });
  } catch {
    return iso;
  }
}

export function EventTicker({ events, isConnected }: EventTickerProps) {
  if (events.length === 0) {
    return (
      <div className="event-ticker">
        <div className="empty-state" style={{ padding: "20px 0" }}>
          <div className="empty-state-icon"><RefreshCw size={24} /></div>
          <div className="empty-state-title">
            {isConnected ? "Waiting for events…" : "No events yet"}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="event-ticker">
      {isConnected && (
        <div className="flex items-center gap-2 mb-4" style={{ marginBottom: 12 }}>
          <span className="live-dot" style={{ width: 6, height: 6 }} />
          <span style={{ fontSize: 11, color: "var(--text-muted)", fontFamily: "var(--font-mono)" }}>
            Live
          </span>
        </div>
      )}
      {events.map((event, i) => (
        <div className="event-item" key={`${event.sequence}-${i}`}>
          <div className="event-item-header">
            <span className={`event-type-chip ${chipClass(event.type)}`}>
              {event.type.replace(/_/g, "_")}
            </span>
            {event.module && (
              <span className="event-module">{event.module}</span>
            )}
            <span className="event-time">{formatTime(event.time)}</span>
          </div>
          {event.message && (
            <PretextBlock
              className="event-message"
              text={event.message}
              font={pretextFonts.eventMessage}
              lineHeight={lineHeights.body}
            />
          )}
        </div>
      ))}
    </div>
  );
}
