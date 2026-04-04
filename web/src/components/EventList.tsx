import { formatDate } from "../lib/format";
import type { ScanEvent } from "../types";
import { EmptyState } from "./EmptyState";

type EventListProps = {
  events: ScanEvent[];
};

export function EventList({ events }: EventListProps) {
  if (events.length === 0) {
    return <EmptyState title="No events yet" detail="Progress and module activity will appear here once the scan emits timeline events." />;
  }

  return (
    <div className="event-list">
      {events.map((event) => (
        <div className="event-row" key={`${event.sequence}-${event.type}`}>
          <div className="event-mark" aria-hidden="true" />
          <div className="event-copy">
            <div className="row-title">
              <strong>{event.type}</strong>
              <span className="chip">{event.module || "scan"}</span>
            </div>
            <p>{event.message || "No message"}</p>
            <span className="mono">{formatDate(event.time)}</span>
          </div>
        </div>
      ))}
    </div>
  );
}
