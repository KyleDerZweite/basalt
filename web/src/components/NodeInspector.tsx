import { Crosshair, ExternalLink } from "lucide-react";
import type { WorkspaceNode } from "../types";
import { formatNodeType } from "../lib/format";
import { PretextBlock } from "./PretextBlock";
import { lineHeights, pretextFonts } from "../lib/typography";

const READABLE_PROPS = new Set([
  "site_name", "profile_url", "full_name", "location",
  "website", "bio", "email", "username", "organization",
  "seed_type",
]);

function confidenceClass(c: number): string {
  if (c >= 0.75) return "high";
  if (c >= 0.5) return "medium";
  return "low";
}

interface NodeInspectorProps {
  node: WorkspaceNode | null;
}

export function NodeInspector({ node }: NodeInspectorProps) {
  if (!node) {
    return (
      <div className="node-inspector">
        <div className="empty-state" style={{ padding: "24px 0" }}>
          <div className="empty-state-icon"><Crosshair size={24} /></div>
          <div className="empty-state-title">No node selected</div>
          <div className="empty-state-desc">Click a node in the graph to inspect it.</div>
        </div>
      </div>
    );
  }

  const confidence = node.confidence ?? 0;
  const props = Object.entries(node).filter(
    ([k, v]) =>
      READABLE_PROPS.has(k) &&
      v != null &&
      v !== "" &&
      k !== "profile_url"
  );

  return (
    <div className="node-inspector">
      <PretextBlock
        className="inspector-label"
        text={node.label}
        font={pretextFonts.inspectorLabel}
        lineHeight={lineHeights.card}
      />

      {/* Badges */}
      <div className="inspector-badges">
        <span className="type-badge">{formatNodeType(node.type)}</span>
        {node.category !== node.type && (
          <span className="type-badge" style={{ borderColor: "var(--accent-dim)", color: "var(--accent)" }}>
            {node.category}
          </span>
        )}
        {node.collapsed_count != null && node.collapsed_count > 0 && (
          <span className="type-badge">+{node.collapsed_count} hidden</span>
        )}
      </div>

      {/* Confidence */}
      {confidence > 0 && (
        <div className="confidence-bar-wrap">
          <div className="confidence-label-row">
            <span>Confidence</span>
            <span>{Math.round(confidence * 100)}%</span>
          </div>
          <div className="confidence-bar">
            <div
              className={`confidence-fill ${confidenceClass(confidence)}`}
              style={{ width: `${confidence * 100}%` }}
            />
          </div>
        </div>
      )}

      {/* Profile link */}
      {node.profile_url && (
        <a
          href={node.profile_url}
          target="_blank"
          rel="noopener noreferrer"
          className="finding-link"
        >
          <ExternalLink size={11} /> View Profile
        </a>
      )}

      {/* Properties table */}
      {props.length > 0 && (
        <div className="props-table">
          {props.map(([key, value]) => (
            <div className="props-row" key={key}>
              <div className="props-key">{key.replace(/_/g, " ")}</div>
              <PretextBlock
                className="props-val"
                text={String(value)}
                font={pretextFonts.inspectorValue}
                lineHeight={lineHeights.body}
                whiteSpace="pre-wrap"
              />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
