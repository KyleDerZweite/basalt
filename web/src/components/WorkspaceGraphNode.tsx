import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";

import { formatNodeCategory, formatNodeType } from "../lib/format";
import type { GraphNodeData } from "../types";

export function WorkspaceGraphNode({ data, selected }: NodeProps<Node<GraphNodeData>>) {
  return (
    <div className={selected ? `workspace-node is-selected ${data.category}` : `workspace-node ${data.category}`}>
      <Handle className="hidden-handle" position={Position.Left} type="target" />
      <div className="workspace-node-head">
        <span className="node-kind">{formatNodeCategory(data.category)}</span>
        <span className="node-type">{formatNodeType(data.type)}</span>
      </div>
      <strong>{data.label}</strong>
      <div className="workspace-node-meta">
        {typeof data.confidence === "number" && data.confidence > 0 ? <span>{Math.round(data.confidence * 100)}%</span> : null}
        {data.collapsedCount ? <span>+{data.collapsedCount} hidden</span> : null}
      </div>
      <Handle className="hidden-handle" position={Position.Right} type="source" />
    </div>
  );
}
