import type { RawEdge } from "../types";
import { EmptyState } from "./EmptyState";

type RawEdgesTableProps = {
  edges: RawEdge[];
};

export function RawEdgesTable({ edges }: RawEdgesTableProps) {
  if (edges.length === 0) {
    return <EmptyState title="No raw edges" detail="Edge evidence will appear here when modules emit relationships between raw nodes." />;
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Type</th>
            <th>Source</th>
            <th>Target</th>
            <th>Module</th>
          </tr>
        </thead>
        <tbody>
          {edges.map((edge) => (
            <tr key={edge.id}>
              <td>{edge.type}</td>
              <td className="mono">{edge.source}</td>
              <td className="mono">{edge.target}</td>
              <td>{edge.source_module}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
