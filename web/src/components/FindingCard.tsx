import { ExternalLink } from "lucide-react";
import type { InsightFinding } from "../types";
import { PretextBlock } from "./PretextBlock";
import { lineHeights, pretextFonts } from "../lib/typography";

interface FindingCardProps {
  finding: InsightFinding;
  onSelectNode?: (nodeId: string) => void;
}

export function FindingCard({ finding, onSelectNode }: FindingCardProps) {
  const handleClick = () => {
    if (finding.node_ids?.[0] && onSelectNode) {
      onSelectNode(finding.node_ids[0]);
    }
  };

  return (
    <div
      className="finding-card"
      onClick={finding.node_ids?.length ? handleClick : undefined}
      style={finding.node_ids?.length ? { cursor: "pointer" } : undefined}
    >
      <div className="finding-card-head">
        <PretextBlock
          className="finding-title"
          text={finding.title}
          font={pretextFonts.findingTitle}
          lineHeight={lineHeights.ui}
        />
        {finding.confidence != null && (
          <div className="finding-confidence">
            {Math.round(finding.confidence * 100)}%
          </div>
        )}
      </div>
      <PretextBlock
        className="finding-summary"
        text={finding.summary}
        font={pretextFonts.findingSummary}
        lineHeight={lineHeights.body}
      />
      {finding.profile_url && (
        <a
          href={finding.profile_url}
          target="_blank"
          rel="noopener noreferrer"
          className="finding-link"
          onClick={(e) => e.stopPropagation()}
        >
          <ExternalLink size={11} /> Open
        </a>
      )}
    </div>
  );
}
