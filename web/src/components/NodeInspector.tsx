import { Crosshair, ExternalLink } from "lucide-react";
import type { WorkspaceEdge, WorkspaceNode } from "../types";
import { formatNodeType } from "../lib/format";
import { PretextBlock } from "./PretextBlock";
import { lineHeights, pretextFonts } from "../lib/typography";

function confidenceClass(c: number): string {
  if (c >= 0.75) return "high";
  if (c >= 0.5) return "medium";
  return "low";
}

interface NodeInspectorProps {
  node: WorkspaceNode | null;
  nodes: WorkspaceNode[];
  edges: WorkspaceEdge[];
  onSelectNode?: (id: string) => void;
}

export function NodeInspector({ node, nodes, edges, onSelectNode }: NodeInspectorProps) {
  if (!node) {
    return (
      <div className="node-inspector">
        <div className="empty-state" style={{ padding: "24px 0" }}>
          <div className="empty-state-icon"><Crosshair size={24} /></div>
          <div className="empty-state-title">No node selected</div>
          <div className="empty-state-desc">Click a node in the graph to inspect its evidence.</div>
        </div>
      </div>
    );
  }

  const confidence = node.confidence ?? 0;
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const incoming = edges.filter((e) => e.target === node.id);
  const outgoing = edges.filter((e) => e.source === node.id);
  const propEntries = Object.entries(node.properties ?? {})
    .filter(([, v]) => v != null && v !== "")
    .sort(([a], [b]) => a.localeCompare(b));

  const renderNeighbor = (edgeId: string, neighborId: string, direction: "from" | "to") => {
    const neighbor = byId.get(neighborId);
    const label = neighbor?.label ?? neighborId;
    return (
      <button
        key={edgeId}
        type="button"
        className="inspector-link-row"
        onClick={() => onSelectNode?.(neighborId)}
        title={direction === "from" ? "Incoming connection" : "Outgoing connection"}
      >
        <span className="inspector-link-dir">{direction === "from" ? "←" : "→"}</span>
        <span className="inspector-link-label">{label}</span>
        {neighbor && <span className="type-badge">{formatNodeType(neighbor.type)}</span>}
      </button>
    );
  };

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
        {(node.source_modules ?? []).map((mod) => (
          <span className="type-badge" key={mod}>{mod}</span>
        ))}
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

      {/* Pivot state */}
      {(node.wave != null || node.pivot) && (
        <div className="inspector-meta">
          {node.wave != null && <span>Depth {node.wave}</span>}
          {node.pivot && <span>Used for pivoting</span>}
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

      {/* Evidence properties */}
      {propEntries.length > 0 && (
        <div className="props-table">
          {propEntries.map(([key, value]) => (
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

      {/* Connections */}
      {(incoming.length > 0 || outgoing.length > 0) && (
        <div className="flex-col gap-2">
          <div className="section-title">Connections</div>
          <div className="flex-col gap-1">
            {incoming.map((e) => renderNeighbor(e.id, e.source, "from"))}
            {outgoing.map((e) => renderNeighbor(e.id, e.target, "to"))}
          </div>
        </div>
      )}
    </div>
  );
}
