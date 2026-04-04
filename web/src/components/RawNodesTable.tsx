import { formatNodeType } from "../lib/format";
import type { RawNode } from "../types";
import { EmptyState } from "./EmptyState";

type RawNodesTableProps = {
  nodes: RawNode[];
};

export function RawNodesTable({ nodes }: RawNodesTableProps) {
  if (nodes.length === 0) {
    return <EmptyState title="No raw nodes" detail="Raw evidence nodes will appear here once the scan stores graph results." />;
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Label</th>
            <th>Type</th>
            <th>Module</th>
            <th>Confidence</th>
          </tr>
        </thead>
        <tbody>
          {nodes.map((node) => (
            <tr key={node.id}>
              <td>{node.label}</td>
              <td>{formatNodeType(node.type)}</td>
              <td>{node.source_module}</td>
              <td>{Math.round(node.confidence * 100)}%</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
