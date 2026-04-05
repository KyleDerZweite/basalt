interface StatusPillProps {
  status: string;
}

const labels: Record<string, string> = {
  queued:    "Queued",
  verifying: "Verifying",
  running:   "Running",
  completed: "Completed",
  partial:   "Partial",
  failed:    "Failed",
  canceled:  "Canceled",
};

export function StatusPill({ status }: StatusPillProps) {
  const label = labels[status] ?? status;
  return (
    <span className={`status-pill ${status}`}>
      <span className="pill-dot" />
      {label}
    </span>
  );
}
