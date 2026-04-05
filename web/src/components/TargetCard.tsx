import { X } from "lucide-react";
import type { Target, TargetAlias } from "../types";
import { PretextBlock } from "./PretextBlock";
import { lineHeights, pretextFonts } from "../lib/typography";

interface TargetCardProps {
  target: Target;
  selected?: boolean;
  onClick: () => void;
  onDeleteAlias?: (aliasId: string) => void;
  showAliasDelete?: boolean;
}

export function TargetCard({ target, selected, onClick, onDeleteAlias, showAliasDelete }: TargetCardProps) {
  const aliases: TargetAlias[] = target.aliases ?? [];

  return (
    <div
      className={`target-card${selected ? " selected" : ""}`}
      onClick={onClick}
    >
      <PretextBlock
        className="target-card-name"
        text={target.display_name}
        font={pretextFonts.targetName}
        lineHeight={lineHeights.card}
      />

      {target.notes && (
        <PretextBlock
          className="target-card-notes"
          text={target.notes}
          font={pretextFonts.targetNotes}
          lineHeight={lineHeights.body}
        />
      )}

      {aliases.length > 0 && (
        <div className="alias-chips">
          {aliases.map((alias) => (
            <span className="alias-chip" key={alias.id}>
              <span className="alias-chip-type">{alias.seed_type}</span>
              {alias.seed_value}
              {showAliasDelete && onDeleteAlias && (
                <button
                  className="alias-chip-remove"
                  onClick={(e) => { e.stopPropagation(); onDeleteAlias(alias.id); }}
                  title="Remove alias"
                >
                  <X size={12} />
                </button>
              )}
            </span>
          ))}
        </div>
      )}

      {aliases.length === 0 && (
        <div style={{ fontSize: 12, color: "var(--text-muted)" }}>No aliases yet</div>
      )}
    </div>
  );
}
