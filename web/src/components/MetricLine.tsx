import type { ReactNode } from "react";

type MetricLineProps = {
  label: string;
  value: ReactNode;
  note: string;
};

export function MetricLine({ label, value, note }: MetricLineProps) {
  return (
    <article className="metric-line">
      <span>{label}</span>
      <strong>{value}</strong>
      <p>{note}</p>
    </article>
  );
}
