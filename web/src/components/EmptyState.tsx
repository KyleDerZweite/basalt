import type { ReactNode } from "react";
import { CircleDashed } from "lucide-react";
import { PretextBlock } from "./PretextBlock";
import { lineHeights, pretextFonts } from "../lib/typography";

interface EmptyStateProps {
  icon?: ReactNode;
  title: string;
  desc?: string;
  action?: ReactNode;
}

export function EmptyState({ icon = <CircleDashed size={24} />, title, desc, action }: EmptyStateProps) {
  return (
    <div className="empty-state">
      <div className="empty-state-icon">{icon}</div>
      <PretextBlock
        className="empty-state-title"
        text={title}
        font={pretextFonts.emptyTitle}
        lineHeight={lineHeights.card}
      />
      {desc && (
        <PretextBlock
          className="empty-state-desc"
          text={desc}
          font={pretextFonts.emptyDescription}
          lineHeight={lineHeights.body}
        />
      )}
      {action}
    </div>
  );
}
