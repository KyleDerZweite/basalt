import type { InsightFinding } from "../types";
import { EmptyState } from "./EmptyState";

type FindingListProps = {
  findings: InsightFinding[];
  onSelect: (finding: InsightFinding) => void;
};

export function FindingList({ findings, onSelect }: FindingListProps) {
  if (findings.length === 0) {
    return <EmptyState title="No top findings yet" detail="Basalt will surface the strongest synthesized results here as the scan develops." />;
  }

  return (
    <div className="finding-list">
      {findings.map((finding, index) => (
        <button className="finding-row" key={`${finding.title}-${index}`} onClick={() => onSelect(finding)} type="button">
          <div className="finding-copy">
            <div className="row-title">
              <strong>{finding.title}</strong>
              <span className="chip">{finding.category ?? "finding"}</span>
            </div>
            <p>{finding.summary}</p>
          </div>
          <span className="finding-arrow">Inspect</span>
        </button>
      ))}
    </div>
  );
}
